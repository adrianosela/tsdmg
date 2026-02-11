// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package types

import "io"

type CreateRecordsInput struct {
	Records []Record `json:"records"`
}

func (i *CreateRecordsInput) Write(writer io.Writer) error {
	return (&encoder[CreateRecordsInput]{inner: i}).write(writer)
}

func (i *CreateRecordsInput) Read(reader io.Reader) error {
	return (&encoder[CreateRecordsInput]{inner: i}).read(reader)
}
