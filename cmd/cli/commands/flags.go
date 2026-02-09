// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package commands

var (
	serverURL string

	commonName  string
	sans        []string
	keyOutPath  string
	certOutPath string

	recordType  string
	recordFQDN  string
	recordValue string
	recordTTL   uint32
)
