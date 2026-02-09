// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/adrianosela/tsdmg/pkg/models"
	"github.com/adrianosela/tsdmg/pkg/service/handler/util"
	"github.com/libdns/libdns"
	"go.uber.org/zap"
)

func dnsRecordsDeleteHandler(
	logger *zap.Logger,
	dnsProvider dns.Provider,
) http.Handler {
	postDNSRecordsDeleteHandler := dnsRecordsDeletePOSTHandler(logger, dnsProvider)

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

		deletedRecords := []models.Record{}
		erroredZones := []error{}
		for zone, records := range libdnsRecords {
			created, err := dnsProvider.DeleteRecords(r.Context(), zone, records)
			if err != nil {
				erroredZones = append(
					erroredZones,
					fmt.Errorf("failed to create records in zone \"%s\": %v", zone, err),
				)
				continue
			}
			for _, libdnsRecord := range created {
				deletedRecords = append(deletedRecords, *util.LibDNSToModel(libdnsRecord, zone))
			}
		}

		errMsg := ""
		if err := errors.Join(erroredZones...); err != nil {
			errMsg = err.Error()
		}
		out := &models.DeleteRecordsOutput{
			Records: deletedRecords,
			Error:   errMsg,
		}
		if err := out.Write(w); err != nil {
			logger.Error("failed to encode reqsponse as JSON", zap.Error(err))
			respondError(logger, w, "an unknown error occured... try again later.", http.StatusInternalServerError)
			return
		}
	})
}
