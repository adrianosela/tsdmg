// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package issuer

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"tailscale.com/util/set"
)

func generateTestCA(t *testing.T) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, err
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return nil, nil, err
	}

	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, nil, err
	}

	return cert, priv, nil
}

func createTestCSR(cn string, sans []string) (*x509.CertificateRequest, error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: cn,
		},
		DNSNames: sans,
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, priv)
	if err != nil {
		return nil, err
	}

	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		return nil, err
	}

	return csr, nil
}

func TestSignCSR_Success(t *testing.T) {
	caCert, caKey, err := generateTestCA(t)
	if err != nil {
		t.Fatalf("failed to generate test CA: %v", err)
	}

	iss := NewLocal(caCert, caKey)

	cn := "example.com"
	sans := []string{"www.example.com", "api.example.com"}
	csr, err := createTestCSR(cn, sans)
	if err != nil {
		t.Fatalf("failed to create test CSR: %v", err)
	}

	allowedNames := make(set.Set[string])
	allowedNames.AddSlice([]string{"example.com", "www.example.com", "api.example.com"})
	certs, err := iss.Issue(t.Context(), csr)
	if err != nil {
		t.Fatalf("failed to sign CSR: %v", err)
	}

	cert, err := x509.ParseCertificate(certs[0])
	if err != nil {
		t.Fatalf("failed to parse leaf cert as x509.Certificate: %v", err)
	}

	if cert.Subject.CommonName != cn {
		t.Errorf("expected CN=%s, got %s", cn, cert.Subject.CommonName)
	}

	if len(cert.DNSNames) != len(sans) {
		t.Errorf("expected %d SANs, got %d", len(sans), len(cert.DNSNames))
	}

	for i, san := range sans {
		if cert.DNSNames[i] != san {
			t.Errorf("expected SAN[%d]=%s, got %s", i, san, cert.DNSNames[i])
		}
	}

	// Verify certificate is signed by CA
	if err := cert.CheckSignatureFrom(caCert); err != nil {
		t.Errorf("certificate signature verification failed: %v", err)
	}

	// Verify it's not a CA
	if cert.IsCA {
		t.Error("certificate should not be a CA")
	}

	// Verify key usage
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Error("certificate should have digital signature key usage")
	}
}

func TestSignCSR_InvalidCSR(t *testing.T) {
	caCert, caKey, err := generateTestCA(t)
	if err != nil {
		t.Fatalf("failed to generate test CA: %v", err)
	}

	issuer := NewLocal(caCert, caKey)

	// Invalid CSR
	_, err = issuer.Issue(t.Context(), &x509.CertificateRequest{})
	if err == nil {
		t.Fatal("expected error for invalid PEM, got nil")
	}
}
