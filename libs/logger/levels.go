package logger

import "golang.org/x/exp/slog"

const (
	LevelTrace slog.Level = -8
	LevelDebug slog.Level = -4
	LevelInfo  slog.Level = 0
	LevelWarn  slog.Level = 4
	LevelError slog.Level = 8

	// LevelDisabled means nothing is ever logged (no call site may use this level).
	LevelDisabled slog.Level = 16
)
