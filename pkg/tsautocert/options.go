// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package tsautocert

import (
	"crypto"
	"errors"

	"github.com/adrianosela/tsdmg"
	"github.com/adrianosela/tsdmg/pkg/logger"
	"golang.org/x/crypto/acme/autocert"
)

var (
	errNilLogger      = errors.New("logger must not be nil")
	errNilCache       = errors.New("cache must not be nil")
	errNilTSDMGClient = errors.New("tsdmg (client) must not be nil")
	errEmptyCN        = errors.New("commonName must not be empty")

	errCertificateManagerClosed = errors.New("certificate manager is closed")
)

type config struct {
	logger logger.Logger

	tsdmgClient tsdmg.Client

	certCN   string
	certSANs []string

	acmeAccountKey crypto.Signer
	acmeContact    []string

	cache autocert.Cache
}

func (c *config) validate() error {
	if c.logger == nil {
		return errNilLogger
	}
	if c.cache == nil {
		return errNilCache
	}
	if c.tsdmgClient == nil {
		return errNilTSDMGClient
	}
	if c.certCN == "" {
		return errEmptyCN
	}
	return nil
}

// Option represents a configuration option for initializing a client.
type Option func(*config)

// WithLogger is a configuration option to configure a logger.
// If this option is not set a log/slog logger is used with a
// JSON handler.
func WithLogger(logger logger.Logger) Option {
	return func(c *config) { c.logger = logger }
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
func WithCertificateCache(cache autocert.Cache) Option {
	return func(c *config) { c.cache = cache }
}

// WithACMEAccountKey is a configuration option to configure an
// ACME account key. If this option is unset, a key will be
// generated at runtime.
func WithACMEAccountKey(acmeAccountKey crypto.Signer) Option {
	return func(c *config) { c.acmeAccountKey = acmeAccountKey }
}

// WithACMEAContact is a configuration option to configure
// contact details for the ACME account e.g.
func WithACMEAContact(acmeContact []string) Option {
	return func(c *config) { c.acmeContact = acmeContact }
}
