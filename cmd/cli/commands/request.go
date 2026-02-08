// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package commands

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adrianosela/tsdmg/pkg/models"
)

var (
	serverURL   string
	commonName  string
	sans        []string
	keyOutPath  string
	certOutPath string
)

var requestCmd = &cobra.Command{
	Use:   "request",
	Short: "Request a certificate from the tsdmg",
	Long: `Generate a private key and CSR, then request a certificate from the Tailscale Domain Management Gateway.
All hostnames (CN and SANs) must be in your CAPS allowlist.`,
	Run: runRequest,
}

func runRequest(cmd *cobra.Command, args []string) {
	fmt.Println("Generating ECDSA P-256 private key...")
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		exitWithError("failed to generate private key: %v", err)
	}

	template := x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: commonName,
		},
		DNSNames: sans,
	}

	fmt.Printf("Creating CSR for CN=%s", commonName)
	if len(sans) > 0 {
		fmt.Printf(" with SANs=%s", strings.Join(sans, ","))
	}
	fmt.Println()

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, privateKey)
	if err != nil {
		exitWithError("failed to create certificate request: %v", err)
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})

	fmt.Printf("Sending CSR to %s...\n", serverURL)
	reqBody := models.CSRRequest{
		CSRPEM: string(csrPEM),
	}
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		exitWithError("failed to marshal request: %v", err)
	}

	resp, err := http.Post(serverURL+"/csr", "application/json", bytes.NewReader(reqJSON))
	if err != nil {
		exitWithError("failed to send request to server: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		exitWithError("failed to read response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp models.CSRResponse
		if err := json.Unmarshal(respBody, &errResp); err == nil && errResp.Error != "" {
			exitWithError("server returned error: %s", errResp.Error)
		}
		exitWithError("server returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var csrResp models.CSRResponse
	if err := json.Unmarshal(respBody, &csrResp); err != nil {
		exitWithError("failed to parse response: %v", err)
	}

	if csrResp.Error != "" {
		exitWithError("server returned error: %s", csrResp.Error)
	}

	fmt.Printf("Writing private key to %s...\n", keyOutPath)
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		exitWithError("failed to marshal private key: %v", err)
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	})

	if err := os.WriteFile(keyOutPath, keyPEM, 0600); err != nil {
		exitWithError("failed to write private key: %v", err)
	}

	fmt.Printf("Writing certificate to %s...\n", certOutPath)
	if err := os.WriteFile(certOutPath, []byte(csrResp.CertificatePEM), 0644); err != nil {
		exitWithError("failed to write certificate: %v", err)
	}

	fmt.Println("Certificate request successful!")
	fmt.Printf("  Private key: %s\n", keyOutPath)
	fmt.Printf("  Certificate: %s\n", certOutPath)
}
