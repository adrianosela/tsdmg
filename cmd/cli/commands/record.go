// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package commands

import (
	"fmt"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/service"
	"github.com/adrianosela/tsdmg/pkg/types"
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

var recordDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a DNS record",
	Run:   recordDeleteHandler,
}

func recordCreateHandler(cmd *cobra.Command, args []string) {
	cl := service.NewClient(http.DefaultClient, serverURL)

	record := types.Record{
		Type:  recordType,
		FQDN:  recordFQDN,
		Value: recordValue,
		TTL:   recordTTL,
	}
	input := &types.CreateRecordsInput{
		Records: []types.Record{record},
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

func recordDeleteHandler(cmd *cobra.Command, args []string) {
	cl := service.NewClient(http.DefaultClient, serverURL)

	record := types.Record{
		Type:  recordType,
		FQDN:  recordFQDN,
		Value: recordValue,
	}
	input := &types.DeleteRecordsInput{
		Records: []types.Record{record},
	}

	output, err := cl.DeleteRecords(cmd.Context(), input)
	if err != nil {
		exitWithError("failed to create record: %v", err)
	}
	if output.Error != "" {
		exitWithError("server failed to create record: %s", output.Error)
	}

	fmt.Println("DNS record deleted successfully!")
}
