// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/authorizer"
	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/adrianosela/tsdmg/pkg/issuer"
	"github.com/adrianosela/tsdmg/pkg/service/handler"
	"go.uber.org/zap"
	"tailscale.com/client/local"
)

const (
	csrCapName = "tsdmg.net/csr/v1"
	dnsCapName = "tsdmg.net/dns/v1"
)

var (
	errNilLogger = errors.New("logger must not be nil")
)

type config struct {
	logger         *zap.Logger
	tsClient       *local.Client
	dnsProvider    dns.Provider
	acmeAccountKey crypto.Signer
	acmeContact    []string
}

func (c *config) validate() error {
	if c.logger == nil {
		return errNilLogger
	}
	return nil
}

type Service struct {
	handler http.Handler
}

func New(
	ctx context.Context,
	tsClient *local.Client,
	dnsProvider dns.Provider,
	opts ...Option,
) (*Service, error) {
	cfg := &config{
		logger:         zap.NewNop(),
		tsClient:       tsClient,
		dnsProvider:    dnsProvider,
		acmeAccountKey: nil,
		acmeContact:    []string{},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	if cfg.acmeAccountKey == nil {
		accountKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("no acme account key was provided and failed to generate one: %v", err)
		}
		cfg.acmeAccountKey = accountKey
	}

	authzer := authorizer.NewDNS(cfg.logger, cfg.tsClient, dnsCapName)
	certIssuer, err := issuer.NewACME(ctx, cfg.dnsProvider, cfg.acmeAccountKey, cfg.acmeContact...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize acme certiticate issuer: %v", err)
	}

	return &Service{
		handler: handler.GetHandler(
			cfg.logger,
			cfg.dnsProvider,
			authzer,
			certIssuer,
		),
	}, nil
}

func (s *Service) ServeHTTP(ln net.Listener) error {
	return http.Serve(ln, s.handler)
}
