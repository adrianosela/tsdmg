// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package authorizer

import (
	"context"
	"crypto/x509"
	"fmt"

	"tailscale.com/client/local"
	"tailscale.com/tailcfg"
	"tailscale.com/util/set"
)

type cap struct {
	Subjects []string `json:"subjects"`
}

type Authorizer struct {
	tsClient *local.Client
	capKey   string
}

type Result struct {
	UserProfile      *tailcfg.UserProfile
	Allowed          bool
	NotAllowedReason string
}

func New(
	tsClient *local.Client,
	capKey string,
) *Authorizer {
	return &Authorizer{
		tsClient: tsClient,
		capKey:   capKey,
	}
}

func (a *Authorizer) Authorize(
	ctx context.Context,
	csr *x509.CertificateRequest,
	remoteAddr string,
) (*Result, error) {
	who, err := a.tsClient.WhoIs(ctx, remoteAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve whois data for remote client: %v", err)
	}

	caps, err := tailcfg.UnmarshalCapJSON[cap](who.CapMap, tailcfg.PeerCapability(a.capKey))
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal capabilities JSON: %v", err)
	}

	allowedNames := make(set.Set[string])
	for _, cap := range caps {
		allowedNames.AddSlice(cap.Subjects)
	}
	if allowedNames.Len() == 0 {
		return &Result{
			UserProfile:      who.UserProfile,
			Allowed:          false,
			NotAllowedReason: "user has no allowed subjects in caps",
		}, nil
	}

	if err := csrIsAllowed(csr, allowedNames); err != nil {
		return &Result{
			UserProfile:      who.UserProfile,
			Allowed:          false,
			NotAllowedReason: err.Error(),
		}, nil
	}

	return &Result{
		UserProfile: who.UserProfile,
		Allowed:     true,
	}, nil
}

func csrIsAllowed(csr *x509.CertificateRequest, allowlist set.Set[string]) error {
	requestedNames := make([]string, 0)
	if csr.Subject.CommonName != "" {
		requestedNames = append(requestedNames, csr.Subject.CommonName)
	}
	requestedNames = append(requestedNames, csr.DNSNames...)

	if err := validateNames(requestedNames, allowlist); err != nil {
		return err
	}

	return nil
}

// validateNames checks if all requested names are in the allowlist
func validateNames(requested []string, allowlist set.Set[string]) error {
	if len(requested) == 0 {
		return fmt.Errorf("no names requested in CSR")
	}

	unauthorized := make([]string, 0)
	for _, name := range requested {
		if _, ok := allowlist[name]; !ok {
			unauthorized = append(unauthorized, name)
		}
	}

	if len(unauthorized) > 0 {
		return fmt.Errorf("unauthorized names requested: %v (allowed: %v)", unauthorized, allowlist.Slice())
	}

	return nil
}
