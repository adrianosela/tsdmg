// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package tsdmg

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/adrianosela/tsdmg/pkg/logger"
	"tailscale.com/client/local"
)

var (
	errNilLogger        = errors.New("logger must not be nil")
	errEmptyServerURL   = errors.New("serverURL must not be empty")
	errInvalidServerURL = errors.New("serverURL is not a valid URL")

	errClientClosed = errors.New("client is closed")
)

type config struct {
	logger logger.Logger

	serverURL string

	skipTailscaleNode bool
	tailscaleClient   *local.Client
}

func (c *config) validate() error {
	if c.logger == nil {
		return errNilLogger
	}
	if c.serverURL == "" {
		return errEmptyServerURL
	}
	if _, err := url.Parse(c.serverURL); err != nil {
		return fmt.Errorf("%w: %v", errInvalidServerURL, err)
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
