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

// WithRegistration is a configuration option to enable the self
// registration endpoint (POST /dns/v1/register). This endpoint
// ensures the presense of A and AAAA records for the client's
// Tailscale private IPv4 and IPv6 addresses respectively. If this
// option is not set, or the list of domains is empty, the self
// registration feature will be disabled.
//
// Note that ACLs still apply to the registration feature... you
// will need to ensure the client node can manage the A and AAAA
// records for every domain passed here, e.g.
//
// "grants": [
//
//	{
//		  "src": ["*"],
//		  "dst": ["*"],
//		  "ip":  ["*"],
//		  "app": {
//		  	  "tsdmg.net/dns/v1": [
//		  	  	  {
//		  	  	  	  "A":    ["${node}.yourdomain.com"],
//		  	  	  	  "AAAA": ["${node}.yourdomain.com"],
//		  	  	  },
//		  	  ],
//		  },
//	},
//
// ],
func WithRegistration(domains ...string) Option {
	return func(c *config) {
		c.registrationDomains = append([]string{}, domains...)
	}
}
