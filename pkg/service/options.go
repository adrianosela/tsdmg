// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package service

import (
	"github.com/adrianosela/tsdmg/pkg/logger"
)

// Option represents a configuration option for initializing a Service.
type Option func(*config)

// WithLogger is a configuration option to configure a logger.
// If this option is not set a log/slog logger is used with a
// JSON handler.
func WithLogger(logger logger.Logger) Option {
	return func(c *config) { c.logger = logger }
}

// WithDomains is a configuration option to define which DNS
// zones the tsdmg server is allowed to manage. If this option
// is not set, the server assumes it can manage any domain.
//
// This option also helps the tsdmg server determine in which
// zone a requested record will be created in: if the option
// is set, a requested record will be created in the zone in
// this list that has the longest matching suffix to the FQDN
// in the request.
//
// If this option is not set, a requested record will be created
// in a zone inferred from Public Suffix List data.
// See https://publicsuffix.org/ for more info.
func WithDomains(domains ...string) Option {
	return func(c *config) {
		c.domains = append([]string{}, domains...)
	}
}

// WithNodeRegDomains is a configuration option to enable the self
// registration endpoint (POST /dns/v1/register). This endpoint
// ensures the presense of A and AAAA records for the client's
// Tailscale private IPv4 and IPv6 addresses respectively.
//
// If both WithDomains and WithNodeRegDomains are set, the domains
// passed here MUST be a subset of those passed to WithDomains.
//
// If this option is not set, the list of domains from WithDomains
// will be used. If both WithDomains and WithNodeRegDomains are
// not set, the registration endpoint will be disabled.
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
func WithNodeRegDomains(domains ...string) Option {
	return func(c *config) {
		c.nodeRegDomains = append([]string{}, domains...)
	}
}
