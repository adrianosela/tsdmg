// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package types

import (
	"encoding/json"
	"fmt"
	"io"
)

type encoder[T any] struct {
	inner *T
}

func (e *encoder[T]) write(writer io.Writer) error {
	if err := json.NewEncoder(writer).Encode(e.inner); err != nil {
		return fmt.Errorf("failed to encode JSON: %v", err)
	}
	return nil
}

func (e *encoder[T]) read(reader io.Reader) error {
	if err := json.NewDecoder(reader).Decode(e.inner); err != nil {
		return fmt.Errorf("failed to decode JSON: %v", err)
	}
	return nil
}
