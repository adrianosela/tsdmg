// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package dns

import (
	"fmt"
	"strings"

	"golang.org/x/net/publicsuffix"
)

func SplitZone(fqdn string) (zone, name string, err error) {
	fqdn = strings.TrimSuffix(fqdn, ".")

	zone, err = publicsuffix.EffectiveTLDPlusOne(fqdn)
	if err != nil {
		return "", "", err
	}

	if fqdn == zone {
		return zone, "", nil
	}

	name = strings.TrimSuffix(fqdn, fmt.Sprintf(".%s", zone))
	return zone, name, nil
}
