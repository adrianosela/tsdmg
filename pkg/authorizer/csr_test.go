// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package authorizer

import (
	"testing"

	"tailscale.com/util/set"
)

func TestValidateNames(t *testing.T) {
	tests := []struct {
		name        string
		requested   []string
		allowed     set.Set[string]
		shouldError bool
	}{
		{
			name:      "all authorized",
			requested: []string{"example.com", "www.example.com"},
			allowed: set.Set[string]{
				"example.com":     {},
				"www.example.com": {},
				"api.example.com": {},
			},
			shouldError: false,
		},
		{
			name:      "one unauthorized",
			requested: []string{"example.com", "unauthorized.com"},
			allowed: set.Set[string]{
				"example.com": {},
			},
			shouldError: true,
		},
		{
			name:      "empty requested",
			requested: []string{},
			allowed: set.Set[string]{
				"example.com": {},
			},
			shouldError: true,
		},
		{
			name:      "exact match",
			requested: []string{"example.com"},
			allowed: set.Set[string]{
				"example.com": {},
			},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNames(tt.requested, tt.allowed)
			if tt.shouldError && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
		})
	}
}
