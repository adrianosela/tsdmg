// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package certcache

import (
	"context"
	"errors"

	"golang.org/x/crypto/acme/autocert"
)

type Nop struct{}

var errNopCacheImplementation = errors.New("no-op cache implementation")

// NewNop returns a new no-op implementation of Cache.
// It always returns error.
func NewNop() autocert.Cache {
	return &Nop{}
}

func (n *Nop) Put(_ context.Context, _ string, _ []byte) error {
	return errNopCacheImplementation
}

func (n *Nop) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, errNopCacheImplementation
}

func (n *Nop) Delete(_ context.Context, _ string) error {
	return errNopCacheImplementation
}
