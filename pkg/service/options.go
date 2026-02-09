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
