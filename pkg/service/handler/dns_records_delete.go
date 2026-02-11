// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/authorizer"
	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/adrianosela/tsdmg/pkg/logger"
	"github.com/adrianosela/tsdmg/pkg/types"
	"github.com/libdns/libdns"
)

func dnsRecordsDeleteHandler(
	logger logger.Logger,
	dnsProvider dns.Provider,
	dnsAuthorizer *authorizer.DNSAuthorizer,
	zoneAllowlist []string,
) http.Handler {
	postDNSRecordsDeleteHandler := dnsRecordsDeletePOSTHandler(logger, dnsProvider, dnsAuthorizer, zoneAllowlist)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postDNSRecordsDeleteHandler.ServeHTTP(w, r)
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func dnsRecordsDeletePOSTHandler(
	logger logger.Logger,
	dnsProvider dns.Provider,
	dnsAuthorizer *authorizer.DNSAuthorizer,
	zoneAllowlist []string,

) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logger.With(
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		var req types.DeleteRecordsInput
		if err := req.Read(r.Body); err != nil {
			logger.Error("failed to parse request JSON", "error", err)
			respondError(logger, w, "invalid request format", http.StatusBadRequest)
			return
		}

		result, err := dnsAuthorizer.AuthorizeRecords(r.Context(), r.RemoteAddr, req.Records...)
		if err != nil {
			logger.Error("failed to authorize DNS request", "error", err)
			respondGenericInternalServerError(logger, w)
			return
		}
		if !result.Allowed {
			respondError(logger, w, result.NotAllowedReason, http.StatusUnauthorized)
			return
		}

		libdnsRecords := make(map[string][]libdns.Record, len(req.Records))
		for i, requestedRecord := range req.Records {
			record, zone, err := requestedRecord.ToLibDNS(zoneAllowlist...)
			if err != nil {
				errMsg := fmt.Sprintf("failed to convert requested record at index %d to libdns format: %v", i, err)
				respondError(logger, w, errMsg, http.StatusBadRequest)
				return
			}
			if zoneRecords, ok := libdnsRecords[zone]; ok {
				libdnsRecords[zone] = append(zoneRecords, record)
			} else {
				libdnsRecords[zone] = []libdns.Record{record}
			}
		}

		deleted := []types.Record{}
		erroredZones := []error{}
		for zone, recordsToDelete := range libdnsRecords {

			// We need to get the records first because some DNS providers
			// will include a unique ID in the inner record object, required
			// for deletion.
			existingRecords, err := dnsProvider.GetRecords(r.Context(), zone)
			if err != nil {
				erroredZones = append(
					erroredZones,
					fmt.Errorf("failed to get records in zone \"%s\": %v", zone, err),
				)
				continue
			}

			var recordsWithIDs []libdns.Record
			for _, recordToDelete := range recordsToDelete {
				deleteRR := recordToDelete.RR()
				for _, existing := range existingRecords {
					existingRR := existing.RR()
					if existingRR.Type == deleteRR.Type &&
						existingRR.Name == deleteRR.Name &&
						existingRR.Data == deleteRR.Data {
						recordsWithIDs = append(recordsWithIDs, existing)
						break
					}
				}
			}

			if len(recordsWithIDs) == 0 {
				erroredZones = append(
					erroredZones,
					fmt.Errorf("no matching records found to delete in zone \"%s\"", zone),
				)
				continue
			}

			deletedRecords, err := dnsProvider.DeleteRecords(r.Context(), zone, recordsWithIDs)
			if err != nil {
				erroredZones = append(
					erroredZones,
					fmt.Errorf("failed to delete records in zone \"%s\": %v", zone, err),
				)
				continue
			}
			for _, deletedRecord := range deletedRecords {
				deleted = append(deleted, *types.RecordFromLibDNS(deletedRecord, zone))
			}
		}

		errMsg := ""
		if err := errors.Join(erroredZones...); err != nil {
			errMsg = err.Error()
		}
		out := &types.DeleteRecordsOutput{
			Error:   errMsg,
			Records: deleted,
		}
		if err := out.Write(w); err != nil {
			logger.Error("failed to encode reqsponse as JSON", "error", err)
			respondGenericInternalServerError(logger, w)
			return
		}
	})
}
