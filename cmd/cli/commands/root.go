// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cats",
	Short: "CLI for requesting certificates from catsnet CA",
}

func Execute() error {
	rootCmd.AddCommand(requestCmd)

	requestCmd.Flags().StringVarP(&serverURL, "server", "s", "", "CA server URL (required)")
	requestCmd.Flags().StringVarP(&commonName, "cn", "c", "", "Common Name (CN) for the certificate (required)")
	requestCmd.Flags().StringSliceVarP(&sans, "san", "a", []string{}, "Subject Alternative Names (comma-separated)")
	requestCmd.Flags().StringVarP(&keyOutPath, "key-out", "k", "key.pem", "Path to write the private key")
	requestCmd.Flags().StringVarP(&certOutPath, "cert-out", "o", "cert.pem", "Path to write the certificate")

	_ = requestCmd.MarkFlagRequired("server")
	_ = requestCmd.MarkFlagRequired("cn")

	return rootCmd.Execute()
}

func exitWithError(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
