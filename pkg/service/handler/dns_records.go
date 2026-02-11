// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/authorizer"
	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/adrianosela/tsdmg/pkg/types"
	"github.com/libdns/libdns"
	"go.uber.org/zap"
)

func dnsRecordsHandler(
	logger *zap.Logger,
	dnsProvider dns.Provider,
	dnsAuthorizer *authorizer.DNSAuthorizer,
	zoneAllowlist []string,
) http.Handler {
	postDNSRecordsHandler := dnsRecordsPOSTHandler(logger, dnsProvider, dnsAuthorizer, zoneAllowlist)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postDNSRecordsHandler.ServeHTTP(w, r)
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func dnsRecordsPOSTHandler(
	logger *zap.Logger,
	dnsProvider dns.Provider,
	dnsAuthorizer *authorizer.DNSAuthorizer,
	zoneAllowlist []string,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logger.With(
			zap.String("path", r.URL.Path),
			zap.String("remote_addr", r.RemoteAddr),
		)

		var req types.CreateRecordsInput
		if err := req.Read(r.Body); err != nil {
			logger.Error("failed to parse request JSON", zap.Error(err))
			respondError(logger, w, "invalid request format", http.StatusBadRequest)
			return
		}

		result, err := dnsAuthorizer.AuthorizeRecords(r.Context(), r.RemoteAddr, req.Records...)
		if err != nil {
			logger.Error("failed to authorize DNS request", zap.Error(err))
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

		createdRecords := []types.Record{}
		erroredZones := []error{}
		for zone, records := range libdnsRecords {
			created, err := dnsProvider.AppendRecords(r.Context(), zone, records)
			if err != nil {
				erroredZones = append(
					erroredZones,
					fmt.Errorf("failed to create records in zone \"%s\": %v", zone, err),
				)
				continue
			}
			for _, libdnsRecord := range created {
				createdRecords = append(createdRecords, *types.RecordFromLibDNS(libdnsRecord, zone))
			}
		}

		errMsg := ""
		if err := errors.Join(erroredZones...); err != nil {
			errMsg = err.Error()
		}
		out := &types.CreateRecordsOutput{
			Records: createdRecords,
			Error:   errMsg,
		}
		if err := out.Write(w); err != nil {
			logger.Error("failed to encode reqsponse as JSON", zap.Error(err))
			respondGenericInternalServerError(logger, w)
			return
		}
	})
}
