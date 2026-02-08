// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package models

import "io"

type CSRRequest struct {
	CSRPEM string `json:"csr_pem"`
}

func (r *CSRRequest) Write(writer io.Writer) error {
	return (&encoder[CSRRequest]{inner: r}).write(writer)
}

func (r *CSRRequest) Read(reader io.Reader) error {
	return (&encoder[CSRRequest]{inner: r}).read(reader)
}
