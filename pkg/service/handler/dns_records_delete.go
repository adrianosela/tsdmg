// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/authorizer"
	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/adrianosela/tsdmg/pkg/models"
	"github.com/adrianosela/tsdmg/pkg/service/handler/util"
	"github.com/libdns/libdns"
	"go.uber.org/zap"
)

func dnsRecordsDeleteHandler(
	logger *zap.Logger,
	dnsProvider dns.Provider,
	dnsAuthorizer *authorizer.DNSAuthorizer,
) http.Handler {
	postDNSRecordsDeleteHandler := dnsRecordsDeletePOSTHandler(logger, dnsProvider, dnsAuthorizer)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postDNSRecordsDeleteHandler.ServeHTTP(w, r)
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func dnsRecordsDeletePOSTHandler(
	logger *zap.Logger,
	dnsProvider dns.Provider,
	dnsAuthorizer *authorizer.DNSAuthorizer,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logger.With(
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
		)

		var req models.DeleteRecordsInput
		if err := req.Read(r.Body); err != nil {
			logger.Error("failed to parse request JSON", zap.Error(err))
			respondError(logger, w, "invalid request format", http.StatusBadRequest)
			return
		}

		result, err := dnsAuthorizer.AuthorizeRecords(r.Context(), r.RemoteAddr, req.Records...)
		if err != nil {
			logger.Error("failed to authorize DNS request", zap.Error(err))
			respondError(logger, w, "an unknown error occured... try again later.", http.StatusInternalServerError)
			return
		}
		if !result.Allowed {
			respondError(logger, w, result.NotAllowedReason, http.StatusUnauthorized)
			return
		}

		libdnsRecords := make(map[string][]libdns.Record, len(req.Records))
		for i, requestedRecord := range req.Records {
			record, zone, err := util.ModelToLibDNS(requestedRecord)
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

			_, err = dnsProvider.DeleteRecords(r.Context(), zone, recordsWithIDs)
			if err != nil {
				erroredZones = append(
					erroredZones,
					fmt.Errorf("failed to delete records in zone \"%s\": %v", zone, err),
				)
				continue
			}
		}

		errMsg := ""
		if err := errors.Join(erroredZones...); err != nil {
			errMsg = err.Error()
		}
		out := &models.DeleteRecordsOutput{Error: errMsg}
		if err := out.Write(w); err != nil {
			logger.Error("failed to encode reqsponse as JSON", zap.Error(err))
			respondError(logger, w, "an unknown error occured... try again later.", http.StatusInternalServerError)
			return
		}
	})
}
