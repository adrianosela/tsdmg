// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"slices"
	"strings"

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
	errNilLogger           = errors.New("logger must not be nil")
	errNilTsClient         = errors.New("tsClient must not be nil")
	errRegDomainNotAllowed = errors.New("a domain in registration-domains was not present in domains")
)

type config struct {
	logger      *zap.Logger
	tsClient    *local.Client
	dnsProvider dns.Provider
	domains     []string
	regDomains  []string
}

func (c *config) validate() error {
	if c.logger == nil {
		return errNilLogger
	}
	if c.tsClient == nil {
		return errNilTsClient
	}
	if len(c.regDomains) > 0 {
		if len(c.domains) > 0 {
			for _, regDomain := range c.regDomains {
				if !slices.Contains(c.domains, regDomain) {
					return fmt.Errorf(
						"%w: %s not in [ %s ]",
						errRegDomainNotAllowed,
						regDomain,
						strings.Join(c.domains, ", "),
					)
				}
			}
		}
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
		domains:     nil,
		regDomains:  nil,
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &Service{
		handler: handler.GetHandler(
			cfg.logger,
			cfg.tsClient,
			cfg.dnsProvider,
			authorizer.NewDNS(cfg.logger, cfg.tsClient, dnsCapName),
			cfg.domains,
			cfg.regDomains,
		),
	}, nil
}

func (s *Service) ServeHTTP(ln net.Listener) error {
	return http.Serve(ln, s.handler)
}
