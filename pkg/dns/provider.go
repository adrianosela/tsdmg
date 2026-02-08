// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package dns

import "github.com/libdns/libdns"

// Provider represents a DNS provider e.g. GoDaddy, Cloudflare, etc.
// This interface is implemented for all major DNS providers in
// repositories under the github.com/libdns organization.
//
// For example, if using GoDaddy, import "github.com/libdns/godaddy"
// and define a provider as:
//
//	provider := &godaddy.Provider{
//	 	APIToken:  os.Getenv("GODADDY_API_KEY"),
//	 	APISecret: os.Getenv("GODADDY_API_SECRET"),
//	}
type Provider interface {
	libdns.RecordAppender
	libdns.RecordDeleter
}
