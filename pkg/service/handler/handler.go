// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package handler

import (
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/authorizer"
	"github.com/adrianosela/tsdmg/pkg/dns"
	"go.uber.org/zap"
	"tailscale.com/client/local"
)

func GetHandler(
	logger *zap.Logger,
	tsClient *local.Client,
	dnsProvider dns.Provider,
	dnsAuthorizer *authorizer.DNSAuthorizer,
	zoneAllowlist []string,
	regZones []string,
) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/dns/v1/records", dnsRecordsHandler(logger, dnsProvider, dnsAuthorizer, zoneAllowlist))
	mux.Handle("/dns/v1/records/delete", dnsRecordsDeleteHandler(logger, dnsProvider, dnsAuthorizer, zoneAllowlist))
	mux.Handle("/dns/v1/records/register", dnsRecordsRegisterHandler(logger, tsClient, dnsProvider, dnsAuthorizer, zoneAllowlist, regZones))
	return mux
}
