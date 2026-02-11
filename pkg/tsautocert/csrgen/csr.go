// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package csrgen

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
)

var (
	ErrFailedPrivateKeyGen = errors.New("failed to generate private key")
	ErrFailedCSRCreation   = errors.New("failed to create certificate request")
	ErrFailedCSRParsing    = errors.New("failed to parse certificate request as x509.CertificateRequest")
)

func GenerateKeyAndCSR(cn string, sans ...string) (*ecdsa.PrivateKey, *x509.CertificateRequest, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrFailedPrivateKeyGen, err)
	}

	template := x509.CertificateRequest{
		Subject:  pkix.Name{CommonName: cn},
		DNSNames: sans,
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, key)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrFailedCSRCreation, err)
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrFailedCSRParsing, err)
	}

	return key, csr, nil
}
