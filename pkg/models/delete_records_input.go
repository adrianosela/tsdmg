// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package models

import "io"

type DeleteRecordsInput struct {
	Names []string `json:"names"`
}

func (i *DeleteRecordsInput) Write(writer io.Writer) error {
	return (&encoder[DeleteRecordsInput]{inner: i}).write(writer)
}

func (i *DeleteRecordsInput) Read(reader io.Reader) error {
	return (&encoder[DeleteRecordsInput]{inner: i}).read(reader)
}
