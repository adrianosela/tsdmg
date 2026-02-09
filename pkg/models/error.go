// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package models

import "io"

type Error struct {
	Error string `json:"error"`
}

func (e *Error) Write(writer io.Writer) error {
	return (&encoder[Error]{inner: e}).write(writer)
}

func (e *Error) Read(reader io.Reader) error {
	return (&encoder[Error]{inner: e}).read(reader)
}
