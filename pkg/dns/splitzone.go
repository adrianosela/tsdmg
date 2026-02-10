// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package dns

import (
	"fmt"
	"strings"

	"golang.org/x/net/publicsuffix"
)

// SplitZone splits an FQDN into its zone and name parts.
//
// If a zone allowlist is provided, the chosen zone will
// be that of with the longest matching suffix to the fqdn.
//
// If a zone allowlist is not provided, the zone will be
// inferred from Public Suffix List data.
// See https://publicsuffix.org/ for more info.
func SplitZone(fqdn string, zoneAllowlist ...string) (string, string, error) {
	fqdn = strings.TrimSuffix(fqdn, ".")

	var zone string
	if len(zoneAllowlist) > 0 {
		longestSuffixLen := 0
		for _, zoneCandidate := range zoneAllowlist {
			suffixLen := len(zoneCandidate)
			if suffixLen > longestSuffixLen && strings.HasSuffix(fqdn, zoneCandidate) {
				longestSuffixLen = suffixLen
				zone = zoneCandidate
			}
		}
		if longestSuffixLen == 0 {
			return "", "", fmt.Errorf("fqdn %s did not match any allowed zones ([ %s ])", fqdn, strings.Join(zoneAllowlist, ", "))
		}

	} else {
		publicZone, err := publicsuffix.EffectiveTLDPlusOne(fqdn)
		if err != nil {
			return "", "", err
		}
		zone = publicZone
	}
	if fqdn == zone {
		return zone, "", nil
	}

	name := strings.TrimSuffix(fqdn, fmt.Sprintf(".%s", zone))
	return zone, name, nil
}
