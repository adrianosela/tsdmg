// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package issuer

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"time"
)

// localCA is an issuer that issues certificates signed
// with a static certificate and private key.
type localCA struct {
	caCert *x509.Certificate
	caKey  crypto.PrivateKey
}

// NewLocal creates a new local CA issuer.
func NewLocal(caCert *x509.Certificate, caKey crypto.PrivateKey) Issuer {
	return &localCA{
		caCert: caCert,
		caKey:  caKey,
	}
}

// Issue issues a certificate based on the given certificate request (CSR).
func (l *localCA) Issue(ctx context.Context, csr *x509.CertificateRequest) ([][]byte, error) {
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("CSR signature verification failed: %v", err)
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %v", err)
	}

	now := time.Now()
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: csr.Subject.CommonName,
		},
		DNSNames:              csr.DNSNames,
		NotBefore:             now,
		NotAfter:              now.Add(90 * 24 * time.Hour), // 90 day validity
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, l.caCert, csr.PublicKey, l.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %v", err)
	}

	return [][]byte{certDER}, nil
}
