// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package dns

import (
	"testing"
)

func TestSplitZone(t *testing.T) {
	tests := []struct {
		name          string
		fqdn          string
		zoneAllowlist []string
		wantZone      string
		wantName      string
		expectedErr   string
	}{
		// Tests with zone allowlist
		{
			name:          "basic subdomain with allowlist",
			fqdn:          "api.example.com",
			zoneAllowlist: []string{"example.com"},
			wantZone:      "example.com",
			wantName:      "api",
			expectedErr:   "",
		},
		{
			name:          "nested subdomain with allowlist",
			fqdn:          "api.staging.example.com",
			zoneAllowlist: []string{"example.com"},
			wantZone:      "example.com",
			wantName:      "api.staging",
			expectedErr:   "",
		},
		{
			name:          "longest suffix match",
			fqdn:          "api.staging.example.com",
			zoneAllowlist: []string{"example.com", "staging.example.com"},
			wantZone:      "staging.example.com",
			wantName:      "api",
			expectedErr:   "",
		},
		{
			name:          "multiple zones, pick longest",
			fqdn:          "www.api.example.com",
			zoneAllowlist: []string{"com", "example.com", "api.example.com"},
			wantZone:      "api.example.com",
			wantName:      "www",
			expectedErr:   "",
		},
		{
			name:          "fqdn equals zone",
			fqdn:          "example.com",
			zoneAllowlist: []string{"example.com"},
			wantZone:      "example.com",
			wantName:      "",
			expectedErr:   "",
		},
		{
			name:          "fqdn with trailing dot",
			fqdn:          "api.example.com.",
			zoneAllowlist: []string{"example.com"},
			wantZone:      "example.com",
			wantName:      "api",
			expectedErr:   "",
		},
		{
			name:          "no matching zone in allowlist",
			fqdn:          "api.other.com",
			zoneAllowlist: []string{"example.com", "test.com"},
			wantZone:      "",
			wantName:      "",
			expectedErr:   "fqdn api.other.com did not match any allowed zones ([ example.com, test.com ])",
		},
		{
			name:          "empty allowlist entry doesn't match",
			fqdn:          "api.example.com",
			zoneAllowlist: []string{"", "example.com"},
			wantZone:      "example.com",
			wantName:      "api",
			expectedErr:   "",
		},
		// Tests without zone allowlist (using Public Suffix List)
		{
			name:        "public suffix - basic .com",
			fqdn:        "api.example.com",
			wantZone:    "example.com",
			wantName:    "api",
			expectedErr: "",
		},
		{
			name:        "public suffix - nested subdomain",
			fqdn:        "www.api.example.com",
			wantZone:    "example.com",
			wantName:    "www.api",
			expectedErr: "",
		},
		{
			name:        "public suffix - .co.uk",
			fqdn:        "api.example.co.uk",
			wantZone:    "example.co.uk",
			wantName:    "api",
			expectedErr: "",
		},
		{
			name:        "public suffix - .io",
			fqdn:        "test.myapp.io",
			wantZone:    "myapp.io",
			wantName:    "test",
			expectedErr: "",
		},
		{
			name:        "public suffix - apex domain",
			fqdn:        "example.com",
			wantZone:    "example.com",
			wantName:    "",
			expectedErr: "",
		},
		{
			name:        "public suffix - with trailing dot",
			fqdn:        "api.example.com.",
			wantZone:    "example.com",
			wantName:    "api",
			expectedErr: "",
		},
		{
			name:        "public suffix - invalid tld",
			fqdn:        "invalid",
			wantZone:    "",
			wantName:    "",
			expectedErr: "publicsuffix: cannot derive eTLD+1 for domain \"invalid\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			zone, name, err := SplitZone(tt.fqdn, tt.zoneAllowlist...)

			if tt.expectedErr != "" {
				if err == nil {
					t.Errorf("expected error %q, got nil", tt.expectedErr)
					return
				}
				if err.Error() != tt.expectedErr {
					t.Errorf("expected error %q, got %q", tt.expectedErr, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if zone != tt.wantZone {
					t.Errorf("expected zone %q, got %q", tt.wantZone, zone)
				}
				if name != tt.wantName {
					t.Errorf("expected name %q, got %q", tt.wantName, name)
				}
			}
		})
	}
}
