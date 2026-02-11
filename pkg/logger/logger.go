// SPDX-FileCopyrightText: 2026 Adriano Sela Aviles (@adrianosela)
// SPDX-License-Identifier: MIT

package logger

type Logger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Error(msg string, args ...any)
	Fatal(msg string, args ...any)
}
