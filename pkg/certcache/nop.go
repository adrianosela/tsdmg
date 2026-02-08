// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package certcache

import "errors"

type Nop struct{}

// NewNop returns a new no-op implementation of Cache.
// It always returns no-hit on Load(), and always fails
// to persist on Store().
func NewNop() Cache {
	return &Nop{}
}

func (n *Nop) Load() ([]byte, []byte, bool, error) {
	return nil, nil, false, nil
}

func (n *Nop) Store(_ []byte, _ []byte) error {
	return errors.New("no-op cache cannot store")
}
