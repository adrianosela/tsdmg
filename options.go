// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package tsdmg

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/adrianosela/tsdmg/pkg/certcache"
	"go.uber.org/zap"
	"tailscale.com/client/local"
)

var (
	errNilLogger           = errors.New("logger must not be nil")
	errNilCache            = errors.New("cache must not be nil")
	errEmptyCN             = errors.New("commonName must not be empty")
	errEmptyACMEProxyURL   = errors.New("acmeProxyURL must not be empty")
	errInvalidACMEProxyURL = errors.New("acmeProxyURL is not a valid URL")

	errClientClosed = errors.New("client is closed")
)

type config struct {
	logger *zap.Logger
	cache  certcache.Cache

	certCN   string
	certSANs []string

	acmeProxyURL string

	skipTailscaleNode bool
	tailscaleClient   *local.Client
}

func (c *config) validate() error {
	if c.logger == nil {
		return errNilLogger
	}
	if c.cache == nil {
		return errNilCache
	}
	if c.certCN == "" {
		return errEmptyCN
	}
	if c.acmeProxyURL == "" {
		return errEmptyACMEProxyURL
	}
	if _, err := url.Parse(c.acmeProxyURL); err != nil {
		return fmt.Errorf("%w: %v", errInvalidACMEProxyURL, err)
	}
	return nil
}

// Option represents a configuration option for initializing a client.
type Option func(*config)

// WithLogger is a configuration option to configure a logger.
// If this option is not set, a zap.NewNop() logger is used,
// which logs nothing.
func WithLogger(logger *zap.Logger) Option {
	return func(c *config) { c.logger = logger }
}

// WithSkipTailscaleNode is a configuration option to signal
// that the client is an existing Tailscale node, with networking
// already set-up. When this option is set, the client will skip
// initializing an underlying tsnet Tailscale node, and instead
// will rely on the machine's networking allowing it to reach the
// certsnet server a.k.a. acme proxy.
func WithSkipTailscaleNode(skipTailscaleNode bool) Option {
	return func(c *config) { c.skipTailscaleNode = skipTailscaleNode }
}

// WithTailscaleNode is a configuration option to pass an already
// initialized Tailscale (tsnet) client to be used for making
// requests to the certnet server a.k.a. acme proxy.
//
// If WithTailscaleNode and WithSkipTailscaleNode are BOTH unset,
// the client will attempt to initialize a new, ephemeral, tsnet
// Tailscale node. For that to succeed, TS_AUTHKEY env must be set.
func WithTailscaleNode(tailscaleClient *local.Client) Option {
	return func(c *config) { c.tailscaleClient = tailscaleClient }
}

// WithSANs is a configuration option to pass additional hostnames
// to be requested in certificates as SANs.
func WithSANs(sans ...string) Option {
	// NOTE: defensive copying of the given list.
	return func(c *config) { c.certSANs = append([]string{}, sans...) }
}

// WithCertificateCache is a configuration option to configure
// certificate (and private key) caching. If this option is not
// set, the runtime will always attempt to fetch certificates
// from the acme proxy server on start-up, and will be unable to
// persist retrieved certiticates.
//
// It is always a good idea to specify a cache strategy... Or
// the acme proxy will likely hit rate limits for the CN and
// SANS requested.
func WithCertificateCache(cache certcache.Cache) Option {
	return func(c *config) { c.cache = cache }
}
