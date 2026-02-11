// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package types

import "io"

type DeleteRecordsOutput struct {
	Records []Record `json:"records"`
	Error   string   `json:"error,omitempty"`
}

func (o *DeleteRecordsOutput) Write(writer io.Writer) error {
	return (&encoder[DeleteRecordsOutput]{inner: o}).write(writer)
}

func (o *DeleteRecordsOutput) Read(reader io.Reader) error {
	return (&encoder[DeleteRecordsOutput]{inner: o}).read(reader)
}
