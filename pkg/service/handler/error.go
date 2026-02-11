// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package handler

import (
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/logger"
	"github.com/adrianosela/tsdmg/pkg/types"
)

func respondError(logger logger.Logger, w http.ResponseWriter, msg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := (&types.Error{Error: msg}).Write(w); err != nil {
		logger.Error("failed to write error response", "error", err)
	}
}

func respondGenericInternalServerError(logger logger.Logger, w http.ResponseWriter) {
	respondError(logger, w, "an unknown error occurred... try again later.", http.StatusInternalServerError)
}
