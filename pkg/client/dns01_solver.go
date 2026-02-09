// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"fmt"

	"github.com/adrianosela/tsdmg/pkg/dns01"
	"github.com/adrianosela/tsdmg/pkg/models"
	"github.com/adrianosela/tsdmg/pkg/service/handler/util"
	"github.com/libdns/libdns"
)

type DNSProvider struct {
	client Client
}

func NewDNSProvider(client Client) dns01.DNS01Solver {
	return &DNSProvider{client: client}
}

func (p *DNSProvider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	modelRecords := []models.Record{}
	for _, record := range records {
		modelRecords = append(modelRecords, *util.LibDNSToModel(record, zone))
	}
	out, err := p.client.CreateRecords(ctx, &models.CreateRecordsInput{Records: modelRecords})
	if err != nil {
		return nil, fmt.Errorf("failed to create records via tsdmg api: %v", err)
	}
	if out.Error != "" {
		return nil, fmt.Errorf("tsdmg api failed to create records: %s", out.Error)
	}
	libDNSRecords := []libdns.Record{}
	for _, record := range out.Records {
		libDNSRecord, _, err := util.ModelToLibDNS(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record in tsdmg api response to libdns format: %v", err)
		}
		libDNSRecords = append(libDNSRecords, libDNSRecord)
	}
	return libDNSRecords, nil
}

func (p *DNSProvider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	modelRecords := []models.Record{}
	for _, record := range records {
		modelRecords = append(modelRecords, *util.LibDNSToModel(record, zone))
	}
	out, err := p.client.DeleteRecords(ctx, &models.DeleteRecordsInput{Records: modelRecords})
	if err != nil {
		return nil, fmt.Errorf("failed to create records via tsdmg api: %v", err)
	}
	if out.Error != "" {
		return nil, fmt.Errorf("tsdmg api failed to create records: %s", out.Error)
	}
	libDNSRecords := []libdns.Record{}
	for _, record := range out.Records {
		libDNSRecord, _, err := util.ModelToLibDNS(record)
		if err != nil {
			return nil, fmt.Errorf("failed to convert record in tsdmg api response to libdns format: %v", err)
		}
		libDNSRecords = append(libDNSRecords, libDNSRecord)
	}
	return libDNSRecords, nil
}
