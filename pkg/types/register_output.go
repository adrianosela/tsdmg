// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package types

import "io"

type RegisterOutput struct {
	Records []Record `json:"records"`
	Error   string   `json:"error,omitempty"`
}

func (o *RegisterOutput) Write(writer io.Writer) error {
	return (&encoder[RegisterOutput]{inner: o}).write(writer)
}

func (o *RegisterOutput) Read(reader io.Reader) error {
	return (&encoder[RegisterOutput]{inner: o}).read(reader)
}
