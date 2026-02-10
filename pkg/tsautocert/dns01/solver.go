// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package dns01

import (
	"context"
	"fmt"

	"github.com/adrianosela/tsdmg"
	"github.com/adrianosela/tsdmg/pkg/models"
	"github.com/libdns/libdns"
)

type DNS01Solver interface {
	libdns.RecordAppender
	libdns.RecordDeleter
}

type tsdmgDNS01Solver struct {
	client tsdmg.Client
}

func NewTSDMGSolver(client tsdmg.Client) DNS01Solver {
	return &tsdmgDNS01Solver{
		client: client,
	}
}
func (s *tsdmgDNS01Solver) AppendRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	modelRecords := []models.Record{}
	for _, record := range recs {
		modelRecords = append(modelRecords, *models.RecordFromLibDNS(record, zone))
	}

	created, err := s.client.CreateRecords(ctx, modelRecords...)
	if err != nil {
		return nil, fmt.Errorf("failed to create records via tsdmg api: %v", err)
	}

	libDNSRecords := []libdns.Record{}
	for _, record := range created {
		libDNSRecord, _, err := record.ToLibDNS()
		if err != nil {
			return nil, fmt.Errorf("failed to convert record in tsdmg api response to libdns format: %v", err)
		}
		libDNSRecords = append(libDNSRecords, libDNSRecord)
	}

	return libDNSRecords, nil
}

func (s *tsdmgDNS01Solver) DeleteRecords(ctx context.Context, zone string, recs []libdns.Record) ([]libdns.Record, error) {
	modelRecords := []models.Record{}
	for _, record := range recs {
		modelRecords = append(modelRecords, *models.RecordFromLibDNS(record, zone))
	}

	deleted, err := s.client.DeleteRecords(ctx, modelRecords...)
	if err != nil {
		return nil, fmt.Errorf("failed to create records via tsdmg api: %v", err)
	}

	libDNSRecords := []libdns.Record{}
	for _, record := range deleted {
		libDNSRecord, _, err := record.ToLibDNS()
		if err != nil {
			return nil, fmt.Errorf("failed to convert record in tsdmg api response to libdns format: %v", err)
		}
		libDNSRecords = append(libDNSRecords, libDNSRecord)
	}

	return libDNSRecords, nil
}
