// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package models

import "io"

type CSRResponse struct {
	CertificatePEM string `json:"certificate_pem,omitempty"`
	Error          string `json:"error,omitempty"`
}

func (r *CSRResponse) Write(writer io.Writer) error {
	return (&encoder[CSRResponse]{inner: r}).write(writer)
}

func (r *CSRResponse) Read(reader io.Reader) error {
	return (&encoder[CSRResponse]{inner: r}).read(reader)
}
