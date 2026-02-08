// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package certcache

import (
	"fmt"
	"os"
)

type FileSystem struct {
	certPath string
	keyPath  string
}

func NewFileSystemCache(certPath string, keyPath string) (Cache, error) {
	return &FileSystem{
		certPath: certPath,
		keyPath:  keyPath,
	}, nil
}

func (f *FileSystem) Load() ([]byte, []byte, bool, error) {
	certData, err := os.ReadFile(f.certPath)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to read certificate file: %v", err)
	}
	keyData, err := os.ReadFile(f.keyPath)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to read key file: %v", err)
	}
	return certData, keyData, true, nil
}

func (f *FileSystem) Store(certData []byte, keyData []byte) error {
	if err := os.WriteFile(f.certPath, certData, 0o600); err != nil {
		return fmt.Errorf("failed to write certificate file: %v", err)
	}
	if err := os.WriteFile(f.keyPath, keyData, 0o600); err != nil {
		return fmt.Errorf("failed to write key file: %v", err)
	}
	return nil
}
