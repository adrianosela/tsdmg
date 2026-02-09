// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package handler

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/authorizer"
	"github.com/adrianosela/tsdmg/pkg/issuer"
	"github.com/adrianosela/tsdmg/pkg/models"
	"go.uber.org/zap"
)

// NOTE: currently unused... throwaway... for later
func csrHandler(
	logger *zap.Logger,
	csrAuthorizer *authorizer.Authorizer,
	certIssuer issuer.Issuer,
) http.Handler {
	postCSRHandler := csrPOSTHandler(logger, csrAuthorizer, certIssuer)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postCSRHandler.ServeHTTP(w, r)
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func csrPOSTHandler(
	logger *zap.Logger,
	csrAuthorizer *authorizer.Authorizer,
	certIssuer issuer.Issuer,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logger.With(
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
		)

		var req models.CSRRequest
		if err := req.Read(r.Body); err != nil {
			logger.Error("failed to parse request JSON", zap.Error(err))
			respondError(logger, w, "invalid request format", http.StatusBadRequest)
			return
		}

		csrPEM, _ := pem.Decode([]byte(req.CSRPEM))
		if csrPEM == nil {
			logger.Error("failed to decode CSR PEM")
			respondError(logger, w, "failed to decode CSR PEM", http.StatusBadRequest)
			return
		}
		csr, err := x509.ParseCertificateRequest(csrPEM.Bytes)
		if err != nil {
			logger.Error("failed to load CSR PEM data onto x509.CertificateRequest", zap.Error(err))
			respondError(logger, w, "failed to load CSR PEM data onto x509.CertificateRequest", http.StatusBadRequest)
			return
		}

		result, err := csrAuthorizer.Authorize(r.Context(), csr, r.RemoteAddr)
		if err != nil {
			logger.Error("failed to authorize CSR", zap.Error(err))
			respondError(logger, w, "an unknown error occured... try again later.", http.StatusInternalServerError)
			return
		}
		if !result.Allowed {
			respondError(logger, w, result.NotAllowedReason, http.StatusUnauthorized)
			return
		}

		chain, err := certIssuer.Issue(r.Context(), csr)
		if err != nil {
			logger.Error("failed to sign CSR", zap.String("client", result.UserProfile.LoginName), zap.Error(err))
			respondError(logger, w, fmt.Sprintf("failed to sign CSR: %v", err), http.StatusBadRequest)
			return
		}

		logger.Info(
			"signed certificate",
			zap.String("client", result.UserProfile.LoginName),
			zap.String("csr_ca", csr.Subject.CommonName),
			zap.Strings("csr_sans", csr.DNSNames),
		)

		var chainPEM []byte
		for _, certDER := range chain {
			chainPEM = append(chainPEM,
				pem.EncodeToMemory(&pem.Block{
					Type:  "CERTIFICATE",
					Bytes: certDER,
				})...,
			)
		}

		resp := &models.CSRResponse{CertificatePEM: string(chainPEM)}
		if err := resp.Write(w); err != nil {
			logger.Error("failed to encode reqsponse as JSON", zap.Error(err))
			respondError(logger, w, "an unknown error occured... try again later.", http.StatusInternalServerError)
			return
		}
	})
}
