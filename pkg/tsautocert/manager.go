// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package tsautocert

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"slices"
	"sync/atomic"
	"time"

	"github.com/adrianosela/tsdmg"
	"github.com/adrianosela/tsdmg/pkg/certcache"
	"github.com/adrianosela/tsdmg/pkg/csrgen"
	"github.com/adrianosela/tsdmg/pkg/tsautocert/dns01"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

var (
	ErrNoCertificateAvailable   = errors.New("no certificate available")
	ErrCertificateManagerClosed = errors.New("certificate manager is closed")
)

// CertificateManager represents a certificate manager
// that provides automatic access to certificates from
// Let's Encrypt and any other ACME-based CA... similar
// to golang.org/x/crypto/acme/autocert except that this
// package solves the ACME "dns-01" challenge (instead
// of "http-01") by using a tsdmg client to create the
// TXT records required to prove domain ownership.
type CertificateManager struct {
	logger *zap.Logger

	dns01Solver dns01.DNS01Solver

	certCN   string
	certSANs []string

	certReady chan struct{}
	cert      atomic.Pointer[tls.Certificate]

	cache    autocert.Cache
	cacheKey string

	acmeClient *acme.Client

	isOpen atomic.Bool

	refresherCtx     context.Context
	refresherCancel  context.CancelFunc
	refresherStopped chan struct{}
}

func (c *CertificateManager) GetCertificate(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if cert := c.cert.Load(); cert != nil {
		return cert, nil
	}
	return nil, ErrNoCertificateAvailable
}

func (c *CertificateManager) WaitForInitialCert(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.certReady:
		return nil
	}
}

func NewCertificateManager(
	ctx context.Context,
	tsdmg tsdmg.Client,
	commonName string,
	opts ...Option,
) (*CertificateManager, error) {
	cfg := &config{
		logger:         zap.NewNop(),
		tsdmgClient:    tsdmg,
		certCN:         commonName,
		certSANs:       nil,
		acmeAccountKey: nil,
		acmeContact:    []string{},
		cache:          certcache.NewNop(),
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

	if cfg.acmeAccountKey == nil {
		accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("no acme account key was provided and failed to generate one: %v", err)
		}
		cfg.acmeAccountKey = accountKey
	}

	acmeClient := &acme.Client{
		Key:          cfg.acmeAccountKey,
		DirectoryURL: acme.LetsEncryptURL,
		UserAgent:    "tsdmg",
	}
	account := &acme.Account{}
	if len(cfg.acmeContact) > 0 {
		account.Contact = cfg.acmeContact
	}
	if _, err := acmeClient.Register(ctx, account, acme.AcceptTOS); err != nil {
		if err != acme.ErrAccountAlreadyExists {
			return nil, fmt.Errorf("failed to register with acme: %v", err)
		}
	}

	refresherCtx, refresherCancel := context.WithCancel(context.Background())

	cm := &CertificateManager{
		logger: cfg.logger,

		dns01Solver: dns01.NewTSDMGSolver(tsdmg),

		certCN:   cfg.certCN,
		certSANs: cfg.certSANs,

		certReady: make(chan struct{}),
		cert:      atomic.Pointer[tls.Certificate]{},
		cache:     cfg.cache,
		cacheKey:  buildCacheKey(cfg.certCN, cfg.certSANs...),

		acmeClient: acmeClient,

		isOpen: atomic.Bool{},

		refresherCtx:     refresherCtx,
		refresherCancel:  refresherCancel,
		refresherStopped: make(chan struct{}),
	}

	go cm.startRefresher()

	return cm, nil
}

// Close gracefully closes the CertificateManager.
func (c *CertificateManager) Close() error {
	if !c.isOpen.CompareAndSwap(true, false) {
		return errCertificateManagerClosed
	}

	// Stop the certificate refresh go routine.
	c.refresherCancel()
	select {
	case <-c.refresherStopped:
		return nil
	case <-time.After(time.Second * 5):
		return fmt.Errorf("timed out waiting for refresher to stop")
	}
}

func (c *CertificateManager) startRefresher() {
	// Prevent Close() from returning before the refresher routine has exited.
	defer func() { close(c.refresherStopped) }()

	// Get initial certificate from cache
	freshCert := c.tryLoadCertificateFromCache()
	if !freshCert {
		c.refresh()
	}
	close(c.certReady)

	// Log initial cert details
	if initialCert := c.cert.Load(); initialCert != nil {
		if len(initialCert.Certificate) > 0 {
			if parsed, err := x509.ParseCertificate(initialCert.Certificate[0]); err == nil {
				c.logger.Info(
					"certificate ready",
					zap.String("cn", parsed.Subject.CommonName),
					zap.Strings("sans", parsed.DNSNames),
					zap.String("exp", parsed.NotAfter.Format(time.RFC3339)),
				)
			}
		}
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

func (c *CertificateManager) refresh() {
	priv, csr, err := csrgen.GenerateKeyAndCSR(c.certCN, c.certSANs...)
	if err != nil {
		c.logger.Error("failed to generate key and CSR for new certificate", zap.Error(err))
		return
	}

	chain, err := dns01.GetCertificate(
		c.refresherCtx,
		c.logger,
		c.acmeClient,
		c.dns01Solver,
		csr,
		dns01.WithBundle(true),
	)
	if err != nil {
		c.logger.Error("failed to refresh certificate via ACME", zap.Error(err))
		return
	}
	if len(chain) < 1 {
		c.logger.Error("fresh certificate chain has length 0")
		return
	}
	leaf := chain[0]

	cert, err := x509.ParseCertificate(leaf)
	if err != nil {
		c.logger.Error("fresh certificate failed parsing as x509.Certificate", zap.Error(err))
		return
	}

	c.cert.Store(&tls.Certificate{
		Certificate: chain,
		PrivateKey:  priv,
	})
	c.logger.Info(
		"certificate rotated successfully",
		zap.String("cn", cert.Subject.CommonName),
		zap.Strings("sans", cert.DNSNames),
		zap.Time("exp", cert.NotAfter),
	)

	go c.tryPersist(chain, priv)
}

func (c *CertificateManager) tryPersist(chainDER [][]byte, key *ecdsa.PrivateKey) {
	var chainData []byte
	for _, der := range chainDER {
		chainData = append(chainData, pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: der,
		})...)
	}

	if err := c.cache.Put(c.refresherCtx, cacheKeyForCert(c.cacheKey), chainData); err != nil {
		c.logger.Error("failed to persist certificate chain in cache", zap.Error(err))
	}

	keyBytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		c.logger.Error("failed to marshal ECDSA private key", zap.Error(err))
		return
	}
	keyData := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyBytes,
	})

	if err := c.cache.Put(c.refresherCtx, cacheKeyForKey(c.cacheKey), keyData); err != nil {
		c.logger.Error("failed to persist private key in cache", zap.Error(err))
	}
}

func (c *CertificateManager) tryLoadCertificateFromCache() bool {
	chainPEM, err := c.cache.Get(c.refresherCtx, cacheKeyForCert(c.cacheKey))
	if err != nil {
		c.logger.Error("failed to load certificate from cache", zap.Error(err))
		return false
	}

	keyPEM, err := c.cache.Get(c.refresherCtx, cacheKeyForKey(c.cacheKey))
	if err != nil {
		c.logger.Error("failed to load key from cache", zap.Error(err))
		return false
	}

	cert, err := tls.X509KeyPair(chainPEM, keyPEM)
	if err != nil {
		c.logger.Error(
			"failed to materialize cert and key pem data as tls.Certificate",
			zap.String("chain_pem", string(chainPEM)),
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

	if parsed.Subject.CommonName != cn {
		logger.Info(
			"existing certificate common name does not match requested, will refresh",
			zap.String("cert_cn", parsed.Subject.CommonName),
			zap.String("required_cn", cn),
		)
		return true
	}

	// CAs automatically add CN to SANs, so we must expect it there.
	expectedSANs := append([]string{}, sans...)
	if !slices.Contains(sans, cn) {
		expectedSANs = append(sans, cn)
	}
	slices.Sort(expectedSANs)

	if !slices.Equal(parsed.DNSNames, expectedSANs) {
		logger.Info(
			"existing certificate SANs do not match requested, will refresh",
			zap.Strings("cert_sans", parsed.DNSNames),
			zap.Strings("required_sans", expectedSANs),
		)
		return true
	}

	if time.Until(parsed.NotAfter) < threshold {
		logger.Info(
			"existing certificate expires within threshold, will refresh",
			zap.Time("cert_exp", parsed.NotAfter),
			zap.Time("required_min_exp", time.Now().Add(threshold)),
		)
		return true
	}

	return false
}

// buildCacheKey builds a unique cache key for a CN and list of SANS.
// NOTE: the list of SANs must be sorted by the caller.
func buildCacheKey(cn string, sans ...string) string {
	h := sha256.New()
	h.Write([]byte(cn)) // nolint:errcheck
	h.Write([]byte{0})  // nolint:errcheck

	for _, san := range sans {
		h.Write([]byte(san)) // nolint:errcheck
		h.Write([]byte{0})   // nolint:errcheck
	}

	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func cacheKeyForCert(baseKey string) string {
	return fmt.Sprintf("%s-cert", baseKey)
}

func cacheKeyForKey(baseKey string) string {
	return fmt.Sprintf("%s-key", baseKey)
}
