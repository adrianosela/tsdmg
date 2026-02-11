// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package authorizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/adrianosela/tsdmg/pkg/authorizer/wildcard"
	"github.com/adrianosela/tsdmg/pkg/logger"
	"github.com/adrianosela/tsdmg/pkg/types"
	"tailscale.com/client/local"
	"tailscale.com/tailcfg"
	"tailscale.com/util/set"
)

type dnsCap map[string]any

type DNSAuthorizer struct {
	logger   logger.Logger
	tsClient *local.Client
	capKey   string
}

func NewDNS(
	logger logger.Logger,
	tsClient *local.Client,
	csrCapKey string,
) *DNSAuthorizer {
	return &DNSAuthorizer{
		logger:   logger,
		tsClient: tsClient,
		capKey:   csrCapKey,
	}
}

func (a *DNSAuthorizer) AuthorizeRecords(
	ctx context.Context,
	remoteAddr string,
	records ...types.Record,
) (*Result, error) {
	who, err := a.tsClient.WhoIs(ctx, remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve whois data for remote client: %v", err)
	}

	caps, err := tailcfg.UnmarshalCapJSON[dnsCap](who.CapMap, tailcfg.PeerCapability(a.capKey))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal capabilities JSON: %v", err)
	}

	patterns := patternsFromCaps(a.logger, caps)

	for _, record := range records {
		rtypePatterns, ok := patterns[record.Type]
		if !ok {
			return &Result{
				UserProfile:      who.UserProfile,
				Allowed:          false,
				NotAllowedReason: fmt.Sprintf("user has no permissions to manage %s records", record.Type),
			}, nil
		}

		foundMatch := false
		for _, pattern := range rtypePatterns.Slice() {
			// Handle template input.
			pattern = strings.ReplaceAll(pattern, "${node}", who.Node.ComputedName)
			// Wildcard matching (handles no-wildcard matching).
			if wildcard.Match(pattern, record.FQDN) {
				foundMatch = true
				break
			}
		}
		if !foundMatch {
			return &Result{
				UserProfile:      who.UserProfile,
				Allowed:          false,
				NotAllowedReason: fmt.Sprintf("user has no permissions to manage %s records for %s", record.Type, record.FQDN),
			}, nil
		}
	}

	return &Result{
		UserProfile: who.UserProfile,
		Allowed:     true,
	}, nil
}

func patternsFromCaps(logger logger.Logger, caps []dnsCap) map[string]set.Set[string] {
	result := make(map[string]set.Set[string])

	for _, cap := range caps {
		for rtype, patterns := range cap {

			var normalized []string
			switch p := patterns.(type) {
			case string:
				normalized = []string{p}
			case []any:
				for _, v := range p {
					if vStr, ok := v.(string); ok {
						normalized = append(normalized, vStr)
					} else {
						// not parseable, move on
						logger.Warn(
							"got a request from client with invalid capabilities",
							"type", fmt.Sprintf("%T", v),
						)
						continue
					}
				}
			default:
				// not parseable, move on
				logger.Warn(
					"got a request from client with invalid capabilities",
					"type", fmt.Sprintf("%T", patterns),
				)
				continue
			}

			if _, ok := result[rtype]; !ok {
				result[rtype] = set.SetOf(normalized)
			} else {
				result[rtype].AddSlice(normalized)
			}
		}
	}

	return result
}
