// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package handler

import (
	"net/http"

	"github.com/adrianosela/tsdmg/pkg/models"
	"go.uber.org/zap"
)

func respondError(logger *zap.Logger, w http.ResponseWriter, msg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := (&models.Error{Error: msg}).Write(w); err != nil {
		logger.Error("failed to write error response", zap.Error(err))
	}
}

func respondGenericInternalServerError(logger *zap.Logger, w http.ResponseWriter) {
	respondError(logger, w, "an unknown error occurred... try again later.", http.StatusInternalServerError)
}
