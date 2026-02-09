// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package handler

import (
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/authorizer"
	"github.com/adrianosela/tsdmg/pkg/dns"
	"go.uber.org/zap"
)

func GetHandler(
	logger *zap.Logger,
	dnsProvider dns.Provider,
	dnsAuthorizer *authorizer.DNSAuthorizer,
) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/dns/records", dnsRecordsHandler(logger, dnsProvider, dnsAuthorizer))
	mux.Handle("/dns/records/delete", dnsRecordsDeleteHandler(logger, dnsProvider, dnsAuthorizer))
	return mux
}
