package log

import "log/slog"

const (
	LevelTrace slog.Level = -8
	LevelDebug slog.Level = -4
	LevelInfo  slog.Level = 0
	LevelWarn  slog.Level = 4
	LevelError slog.Level = 8

	// LevelDisabled means nothing is ever logged (no call site may use this level).
	LevelDisabled slog.Level = 16
)

// ReplaceLevelAttr rewrites the level attribute to the correct string value.
// This is done because slog doesn't include trace level logging and
// otherwise trace logs show up as `DEBUG-4`.
func ReplaceLevelAttr(groups []string, a slog.Attr) slog.Attr {
	if a.Key != slog.LevelKey {
		return a
	}

	level := a.Value.Any().(slog.Level)
	switch {
	case level < LevelDebug:
		a.Value = slog.StringValue("TRACE")
	case level < LevelInfo:
		a.Value = slog.StringValue("DEBUG")
	case level < LevelWarn:
		a.Value = slog.StringValue("INFO")
	case level < LevelError:
		a.Value = slog.StringValue("WARNING")
	default:
		a.Value = slog.StringValue("ERROR")
	}

	return a
}
