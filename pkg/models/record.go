// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package models

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/adrianosela/tsdmg/pkg/dns"
	"github.com/libdns/libdns"
)

type Record struct {
	Type  string `json:"type"`
	FQDN  string `json:"fqdn"`
	Value string `json:"value"`
	TTL   uint32 `json:"ttl"`
}

func (r *Record) ToLibDNS(zoneAllowlist ...string) (libdns.Record, string, error) {
	zone, name, err := dns.SplitZone(r.FQDN, zoneAllowlist...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to split FQDN into zone and name: %v", err)
	}

	var record libdns.Record
	switch r.Type {
	case "A":
		addr, err := netip.ParseAddr(r.Value)
		if err != nil {
			return nil, "", fmt.Errorf("got an A record request with a non IP address value")
		}
		if !addr.Is4() {
			return nil, "", fmt.Errorf("got an A record request with a non IPv4 address value")
		}
		record = libdns.Address{
			Name: name,
			TTL:  time.Second * time.Duration(r.TTL),
			IP:   addr,
		}
	case "AAAA":
		addr, err := netip.ParseAddr(r.Value)
		if err != nil {
			return nil, "", fmt.Errorf("got an AAAA record request with a non IP address value")
		}
		if !addr.Is6() {
			return nil, "", fmt.Errorf("got an AAAA record request with a non IPv6 address value")
		}
		record = libdns.Address{
			Name: name,
			TTL:  time.Second * time.Duration(r.TTL),
			IP:   addr,
		}
	case "CNAME":
		// Add trailing dot if not already present.
		target := r.Value
		if target != "" && target[len(target)-1] != '.' {
			target = target + "."
		}
		record = libdns.CNAME{
			Name:   name,
			TTL:    time.Second * time.Duration(r.TTL),
			Target: target,
		}
	case "TXT":
		record = libdns.TXT{
			Name: name,
			TTL:  time.Second * time.Duration(r.TTL),
			Text: r.Value,
		}
	default:
		return nil, "", fmt.Errorf("record has invalid type %s: only A, AAAA, CNAME, and TXT are supported", r.Type)
	}

	return record, zone, nil
}

func RecordFromLibDNS(r libdns.Record, zone string) *Record {
	rr := r.RR()
	return &Record{
		Type:  rr.Type,
		FQDN:  fmt.Sprintf("%s.%s", rr.Name, zone),
		Value: rr.Data,
		TTL:   uint32(rr.TTL),
	}
}
