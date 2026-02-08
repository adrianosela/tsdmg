// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package issuer

import (
	"context"
	"crypto/x509"
)

type Issuer interface {
	Issue(ctx context.Context, csr *x509.CertificateRequest) ([][]byte, error)
}
