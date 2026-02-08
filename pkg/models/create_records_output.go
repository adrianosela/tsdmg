// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package models

import "io"

type CreateRecordsOutput struct {
	Records []Record `json:"records"`
	Error   string   `json:"error,omitempty"`
}

func (o *CreateRecordsOutput) Write(writer io.Writer) error {
	return (&encoder[CreateRecordsOutput]{inner: o}).write(writer)
}

func (o *CreateRecordsOutput) Read(reader io.Reader) error {
	return (&encoder[CreateRecordsOutput]{inner: o}).read(reader)
}
