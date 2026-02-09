// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/authorizer"
	"github.com/adrianosela/tsdmg/pkg/dns"
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
	logger      *zap.Logger
	tsClient    *local.Client
	dnsProvider dns.Provider
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
		logger:      zap.NewNop(),
		tsClient:    tsClient,
		dnsProvider: dnsProvider,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	authzer := authorizer.NewDNS(cfg.logger, cfg.tsClient, dnsCapName)

	return &Service{
		handler: handler.GetHandler(
			cfg.logger,
			cfg.dnsProvider,
			authzer,
		),
	}, nil
}

func (s *Service) ServeHTTP(ln net.Listener) error {
	return http.Serve(ln, s.handler)
}
