// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package tsdmg

import (
	"context"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"sync/atomic"
	"time"

	"github.com/adrianosela/tsdmg/pkg/certcache"
	"github.com/adrianosela/tsdmg/pkg/client"
	"github.com/adrianosela/tsdmg/pkg/csrgen"
	"go.uber.org/zap"
	"tailscale.com/client/local"
	"tailscale.com/tsnet"
)

var (
	ErrNoCertAvailable = errors.New("no certificate available")
)

// Client represents a Certsnet Client, capable
// of retrieving certificates as needed
type Client struct {
	logger *zap.Logger

	service client.Client

	certCN   string
	certSANs []string

	certReady chan struct{}
	cert      atomic.Pointer[tls.Certificate]

	cache certcache.Cache

	isOpen  atomic.Bool
	closers []func() error

	refresherCtx     context.Context
	refresherCancel  context.CancelFunc
	refresherStopped chan struct{}
}

func (c *Client) GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if cert := c.cert.Load(); cert != nil {
		return cert, nil
	}
	return nil, ErrNoCertAvailable
}

func (c *Client) WaitForInitialCert(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.certReady:
		return nil
	}
}

func NewClient(
	commonName string,
	acmeProxyURL string,
	opts ...Option,
) (*Client, error) {
	cfg := &config{
		logger:            zap.NewNop(),
		certCN:            commonName,
		certSANs:          nil,
		acmeProxyURL:      acmeProxyURL,
		skipTailscaleNode: false,
		tailscaleClient:   nil,
		cache:             certcache.NewNop(),
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Sort SANs so we can use slices.Equal to check if all SANs
	// are present in existing (cached) certificates.
	slices.Sort(cfg.certSANs)

	// Slice for functions to be called on Client's Close().
	// NOTE: they will be closed in reverse order e.g. LIFO.
	var closers []func() error

	// Initialize tailscale client if none provided via options.
	if cfg.tailscaleClient == nil {
		srv := new(tsnet.Server)
		srv.Ephemeral = true
		if err := srv.Start(); err != nil {
			return nil, fmt.Errorf("failed to start Tailscale tsnet node: %v", err)
		}
		closers = append(closers, srv.Close)

		tsClient, err := srv.LocalClient()
		if err != nil {
			if closeErr := srv.Close(); closeErr != nil {
				cfg.logger.Error("failed to close Tailscale tsnet node")
			}
			return nil, fmt.Errorf("failed to initialize tailscale local client: %v", err)
		}
		cfg.tailscaleClient = tsClient
	}

	httpClient := &http.Client{}
	if cfg.tailscaleClient != nil {
		httpClient = httpClientFromTsClient(cfg.tailscaleClient)
	}

	refresherCtx, refresherCancel := context.WithCancel(context.Background())

	client := &Client{
		logger: cfg.logger,

		service: client.New(httpClient, acmeProxyURL),

		certCN:    cfg.certCN,
		certSANs:  cfg.certSANs,
		certReady: make(chan struct{}),
		cert:      atomic.Pointer[tls.Certificate]{},
		cache:     cfg.cache,

		isOpen:  atomic.Bool{},
		closers: closers,

		refresherCtx:     refresherCtx,
		refresherCancel:  refresherCancel,
		refresherStopped: make(chan struct{}),
	}

	go client.startRefresher()

	return client, nil
}

// Close gracefully closes the Client.
func (c *Client) Close() error {
	if !c.isOpen.CompareAndSwap(true, false) {
		return errClientClosed
	}

	// Stop the certificate refresh go routine.
	c.refresherCancel()
	<-c.refresherStopped

	// Run closers in reverse order.
	errs := []error{}
	for i := len(c.closers) - 1; i >= 0; i-- {
		errs = append(errs, c.closers[i]())
	}

	return errors.Join(errs...)
}

func (c *Client) startRefresher() {
	// Prevent Close() from returning before the refreher routine has exited.
	defer func() { close(c.refresherStopped) }()

	// Get initial certificate from cache
	freshCert := c.tryLoadCertificateFromCache()
	if !freshCert {
		c.refresh()
	}

	interval := time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.refresherCtx.Done():
			return
		case <-ticker.C:
			c.refresh()
			ticker.Reset(interval)
		}
	}
}

func (c *Client) refresh() {
	priv, csr, err := csrgen.GenerateKeyAndCSR(c.certCN, c.certSANs...)
	if err != nil {
		c.logger.Error("failed to generate key and CSR for new certificate", zap.Error(err))
		return
	}
	cert, err := c.service.RequestCertificate(c.refresherCtx, csr)
	if err != nil {
		c.logger.Error("failed to request new certificate from acme proxy", zap.Error(err))
		return
	}

	go c.tryPersist(cert, priv)

	c.cert.Store(&tls.Certificate{
		Certificate: [][]byte{cert.Raw},
		PrivateKey:  priv,
	})
	c.logger.Info(
		"certificate rotated successfully",
		zap.String("cn", cert.Subject.CommonName),
		zap.Strings("sans", cert.DNSNames),
		zap.Time("exp", cert.NotAfter),
	)
}

func (c *Client) tryPersist(cert *x509.Certificate, key *ecdsa.PrivateKey) {
	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		c.logger.Error("failed to marshal ECDSA private key", zap.Error(err))
		return
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})

	if err := c.cache.Store(certPEM, keyPEM); err != nil {
		c.logger.Error("failed to persist fresh certificate in cache", zap.Error(err))
	}
}

func (c *Client) tryLoadCertificateFromCache() bool {
	certPEM, keyPEM, ok, err := c.cache.Load()
	if err != nil {
		c.logger.Error("failed to load certificate from cache", zap.Error(err))
		return false
	}
	if !ok {
		return false
	}

	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		c.logger.Error(
			"failed to materialize cert and key pem data as tls.Certificate",
			zap.String("cert_pem", string(certPEM)),
			zap.Error(err),
		)
		return false
	}

	if needsRefresh(
		c.logger,
		&cert,
		time.Hour*24,
		c.certCN,
		c.certSANs...,
	) {
		return false
	}

	c.cert.Store(&cert)
	return true
}

func needsRefresh(logger *zap.Logger, cert *tls.Certificate, threshold time.Duration, cn string, sans ...string) bool {
	if cert == nil {
		logger.Error("needsRefresh was called with nil certificate")
		return true
	}
	if len(cert.Certificate) < 1 {
		logger.Error("needsRefresh was called with empty certificate")
		return true
	}
	leaf := cert.Certificate[0]

	parsed, err := x509.ParseCertificate(leaf)
	if err != nil {
		logger.Error("existing certificate failed parsing as x509.Certificate", zap.Error(err))
		return true

	}
	slices.Sort(parsed.DNSNames)

	if parsed.Subject.CommonName != cn {
		logger.Info(
			"existing certificate common name does not match requested, will refresh",
			zap.String("cert_cn", parsed.Subject.CommonName),
			zap.String("required_cn", cn),
		)
		return true
	}
	if !slices.Equal(parsed.DNSNames, sans) {
		logger.Info(
			"existing certificate SANs do not match requested, will refresh",
			zap.Strings("cert_sans", parsed.DNSNames),
			zap.Strings("required_sans", sans),
		)
		return true
	}

	if time.Until(parsed.NotAfter) > threshold {
		logger.Info(
			"existing certificate expires within threshold, will refresh",
			zap.Time("cert_exp", parsed.NotAfter),
			zap.Time("required_min_exp", time.Now().Add(threshold)),
		)
		return true
	}

	return false
}

func httpClientFromTsClient(tsClient *local.Client) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return tsClient.Dial(ctx, network, addr)
			},
		},
		// NOTE: request timeouts will be context-controller, no need to define any here...
	}
}
