// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package logger

import (
	"log/slog"
	"os"
)

type slogLogger struct {
	l *slog.Logger
}

func New() Logger {
	return &slogLogger{l: slog.New(slog.NewJSONHandler(os.Stdout, nil))}
}

func (s *slogLogger) Debug(msg string, args ...any) { s.l.Debug(msg, args...) }
func (s *slogLogger) Info(msg string, args ...any)  { s.l.Info(msg, args...) }
func (s *slogLogger) Warn(msg string, args ...any)  { s.l.Warn(msg, args...) }
func (s *slogLogger) Error(msg string, args ...any) { s.l.Error(msg, args...) }
func (s *slogLogger) With(args ...any) Logger       { return &slogLogger{l: s.l.With(args...)} }
