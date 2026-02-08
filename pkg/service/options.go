// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package service

import (
	"go.uber.org/zap"
)

// Option represents a configuration option for initializing a Service.
type Option func(*config)

// WithLogger is a configuration option to configure a logger.
// If this option is not set, a zap.NewNop() logger is used,
// which logs nothing.
func WithLogger(logger *zap.Logger) Option {
	return func(c *config) { c.logger = logger }
}

// WithACMEAccountKey is a configuration option to configure an
// ACME account key. If this option is unset, a key will be
// generated at runtime.
func WithACMEAccountKey(logger *zap.Logger) Option {
	return func(c *config) { c.logger = logger }
}

// WithACMEAContact is a configuration option to configure
// contact details for the ACME account e.g.
func WithACMEAContact(acmeContact []string) Option {
	return func(c *config) { c.acmeContact = acmeContact }
}
