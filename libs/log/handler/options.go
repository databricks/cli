package handler

import "log/slog"

type Options struct {
	// Color enables colorized output.
	Color bool

	// Level is the minimum enabled logging level.
	Level slog.Leveler

	// ReplaceAttr is a function that can be used to replace attributes.
	// Semantics are identical to [slog.ReplaceAttr].
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
}
