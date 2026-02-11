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
	"tailscale.com/client/local"
)

func dnsRecordsRegisterHandler(
	logger logger.Logger,
	tsClient *local.Client,
	dnsProvider dns.Provider,
	dnsAuthorizer *authorizer.DNSAuthorizer,
	zoneAllowlist []string,
	regZones []string,
) http.Handler {
	postDNSRecordsRegisterHandler := dnsRecordsRegisterPOSTHandler(
		logger,
		tsClient,
		dnsProvider,
		dnsAuthorizer,
		zoneAllowlist,
		regZones,
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postDNSRecordsRegisterHandler.ServeHTTP(w, r)
			return
		}

		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	})
}

func dnsRecordsRegisterPOSTHandler(
	logger logger.Logger,
	tsClient *local.Client,
	dnsProvider dns.Provider,
	dnsAuthorizer *authorizer.DNSAuthorizer,
	zoneAllowlist []string,
	regZones []string,
) http.Handler {
	if len(regZones) == 0 {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := logger.With(
				"path", r.URL.Path,
				"remote_addr", r.RemoteAddr,
			)
			respondError(logger, w, "The registration feature is not enabled for this server", http.StatusForbidden)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logger.With(
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		who, err := tsClient.WhoIs(r.Context(), r.RemoteAddr)
		if err != nil {
			logger.Error("failed to retrieve whois data for remote client", "error", err)
			respondGenericInternalServerError(logger, w)
			return
		}

		records := []types.Record{}
		for _, prefix := range who.Node.Addresses {
			addr := prefix.Addr()

			if !addr.IsValid() {
				respondError(logger, w, fmt.Sprintf("client has invalid an address: %s", addr), http.StatusBadRequest)
				return
			}

			if addr.Is4() {
				for _, zone := range regZones {
					records = append(records, types.Record{
						Type:  "A",
						FQDN:  fmt.Sprintf("%s.%s", who.Node.ComputedName, zone),
						Value: addr.String(),
						TTL:   60,
					})
				}
				continue
			}
			if addr.Is6() {
				for _, zone := range regZones {
					records = append(records, types.Record{
						Type:  "AAAA",
						FQDN:  fmt.Sprintf("%s.%s", who.Node.ComputedName, zone),
						Value: addr.String(),
						TTL:   60,
					})
				}
				continue
			}

			respondError(logger, w, fmt.Sprintf("client has invalid an address: %s", addr), http.StatusBadRequest)
			return
		}

		result, err := dnsAuthorizer.AuthorizeRecords(r.Context(), r.RemoteAddr, records...)
		if err != nil {
			logger.Error("failed to authorize DNS request", "error", err)
			respondGenericInternalServerError(logger, w)
			return
		}
		if !result.Allowed {
			respondError(logger, w, result.NotAllowedReason, http.StatusUnauthorized)
			return
		}

		libdnsRecords := make(map[string][]libdns.Record, len(records))
		for i, requestedRecord := range records {
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
			// Fetch existing records in the zone.
			existingRecords, err := dnsProvider.GetRecords(r.Context(), zone)
			if err != nil {
				erroredZones = append(
					erroredZones,
					fmt.Errorf("failed to get existing records in zone \"%s\": %v", zone, err),
				)
				continue
			}

			// Filter out already existing records.
			recordsToCreate := []libdns.Record{}
			for _, record := range records {
				recordRR := record.RR()
				exists := false
				for _, existing := range existingRecords {
					existingRR := existing.RR()
					if existingRR.Type == recordRR.Type && existingRR.Name == recordRR.Name && existingRR.Data == recordRR.Data {
						exists = true
						break
					}
				}
				if !exists {
					recordsToCreate = append(recordsToCreate, record)
				}
			}
			if len(recordsToCreate) == 0 {
				// Move on to next zone
				continue
			}

			created, err := dnsProvider.AppendRecords(r.Context(), zone, recordsToCreate)
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
		out := &types.RegisterOutput{
			Records: createdRecords,
			Error:   errMsg,
		}
		if err := out.Write(w); err != nil {
			logger.Error("failed to encode reqsponse as JSON", "error", err)
			respondGenericInternalServerError(logger, w)
			return
		}
	})
}
