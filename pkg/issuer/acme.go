// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package issuer

import (
	"context"
	"crypto"
	"crypto/x509"
	"fmt"

	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/adrianosela/tsdmg/pkg/dns01"
	"go.uber.org/zap"
	"golang.org/x/crypto/acme"
)

// acmeIssuer is an issuer that issues Let's Encrypt certificates.
type acmeIssuer struct {
	logger      *zap.Logger
	acmeClient  *acme.Client
	dnsProvider dns.Provider
}

// NewACME creates a new ACME issuer.
func NewACME(
	ctx context.Context,
	logger *zap.Logger,
	dnsProvider dns.Provider,
	accountKey crypto.Signer,
	contact ...string,
) (Issuer, error) {

	acmeClient := &acme.Client{
		Key:          accountKey,
		DirectoryURL: acme.LetsEncryptURL,
		UserAgent:    "tsdmg",
	}

	account := &acme.Account{}
	if len(contact) > 0 {
		account.Contact = contact
	}
	if _, err := acmeClient.Register(ctx, account, acme.AcceptTOS); err != nil {
		if err != acme.ErrAccountAlreadyExists {
			return nil, fmt.Errorf("failed to register with acme: %v", err)
		}
	}

	return &acmeIssuer{
		logger:      logger,
		acmeClient:  acmeClient,
		dnsProvider: dnsProvider,
	}, nil
}

// Issue issues a certificate based on the given certificate request (CSR).
func (a *acmeIssuer) Issue(
	ctx context.Context,
	csr *x509.CertificateRequest,
) ([][]byte, error) {
	return dns01.GetCertificate(ctx, a.logger, a.acmeClient, a.dnsProvider, csr)
}
