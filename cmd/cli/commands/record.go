// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package commands

import (
	"fmt"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/client"
	"github.com/adrianosela/tsdmg/pkg/models"
	"github.com/spf13/cobra"
)

var recordCmd = &cobra.Command{
	Use:   "record",
	Short: "DNS management commands",
	Long:  "Request a DNS record to be created/updated/deleted via the Tailscale Domain Management Gateway.",
}

var recordCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a DNS record",
	Run:   recordCreateHandler,
}

func recordCreateHandler(cmd *cobra.Command, args []string) {
	cl := client.New(http.DefaultClient, serverURL)

	record := models.Record{
		Type:  recordType,
		FQDN:  recordFQDN,
		Value: recordValue,
		TTL:   recordTTL,
	}
	input := &models.CreateRecordsInput{
		Records: []models.Record{record},
	}

	output, err := cl.CreateRecords(cmd.Context(), input)
	if err != nil {
		exitWithError("failed to create record: %v", err)
	}
	if output.Error != "" {
		exitWithError("server failed to create record: %s", output.Error)
	}

	if len(output.Records) > 0 {
		fmt.Println("DNS record created successfully!")
		fmt.Printf("  Type: %s\n", output.Records[0].Type)
		fmt.Printf("  Name: %s\n", output.Records[0].FQDN)
		fmt.Printf("  Value: %s\n", output.Records[0].Value)
		fmt.Printf("  TTL: %d\n", output.Records[0].TTL)
	}
}
