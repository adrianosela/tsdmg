// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package certcache

type Cache interface {
	Store([]byte, []byte) error
	Load() ([]byte, []byte, bool, error)
}
