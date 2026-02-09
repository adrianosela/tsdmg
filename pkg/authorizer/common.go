// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package authorizer

import "tailscale.com/tailcfg"

type Result struct {
	UserProfile      *tailcfg.UserProfile
	Allowed          bool
	NotAllowedReason string
}
