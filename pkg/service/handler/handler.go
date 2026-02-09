// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package handler

import (
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/authorizer"
	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/adrianosela/tsdmg/pkg/issuer"
	"go.uber.org/zap"
)

func GetHandler(
	logger *zap.Logger,
	dnsProvider dns.Provider,
	csrAuthorizer *authorizer.Authorizer,
	certIssuer issuer.Issuer,
) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/dns/records", dnsRecordsHandler(logger, dnsProvider))
	mux.Handle("/dns/records/delete", dnsRecordsDeleteHandler(logger, dnsProvider))
	return mux
}
