// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tssl",
	Short: "CLI for requesting records from a tsdmg server",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
}

func Execute() error {
	recordCreateCmd.Flags().StringVarP(&serverURL, "server", "s", "", "TSDMG server URL (required)")
	recordCreateCmd.Flags().StringVarP(&recordType, "type", "t", "", "DNS record type (required)")
	recordCreateCmd.Flags().StringVarP(&recordFQDN, "fqdn", "f", "", "DNS record FQDN (required)")
	recordCreateCmd.Flags().StringVarP(&recordValue, "value", "v", "", "DNS record value (required)")
	recordCreateCmd.Flags().Uint32VarP(&recordTTL, "ttl", "d", 60, "DNS Record TTL in seconds")
	_ = recordCreateCmd.MarkFlagRequired("server")
	_ = recordCreateCmd.MarkFlagRequired("type")
	_ = recordCreateCmd.MarkFlagRequired("fqdn")
	_ = recordCreateCmd.MarkFlagRequired("value")
	recordCmd.AddCommand(recordCreateCmd)

	recordDeleteCmd.Flags().StringVarP(&serverURL, "server", "s", "", "TSDMG server URL (required)")
	recordDeleteCmd.Flags().StringVarP(&recordType, "type", "t", "", "DNS record type (required)")
	recordDeleteCmd.Flags().StringVarP(&recordFQDN, "fqdn", "f", "", "DNS record FQDN (required)")
	recordDeleteCmd.Flags().StringVarP(&recordValue, "value", "v", "", "DNS record value (required)")
	_ = recordDeleteCmd.MarkFlagRequired("server")
	_ = recordDeleteCmd.MarkFlagRequired("type")
	_ = recordDeleteCmd.MarkFlagRequired("fqdn")
	_ = recordDeleteCmd.MarkFlagRequired("value")
	recordCmd.AddCommand(recordDeleteCmd)

	rootCmd.AddCommand(recordCmd)
	return rootCmd.Execute()
}

func exitWithError(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+msg+"\n", args...)
	os.Exit(1)
}
