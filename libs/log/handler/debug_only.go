package handler

import (
	"context"
	"log/slog"

	"github.com/databricks/cli/libs/log"
)

// DebugOnlyHandler logs records above Debug (e.g. hostmetadata's "failed to
// fetch host metadata" Warn) at Debug, so they show only with --debug.
type DebugOnlyHandler struct{ slog.Handler }

func debugOnly(level slog.Level) slog.Level {
	return min(level, log.LevelDebug)
}

func (h DebugOnlyHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.Handler.Enabled(ctx, debugOnly(level))
}

func (h DebugOnlyHandler) Handle(ctx context.Context, r slog.Record) error {
	r.Level = debugOnly(r.Level)
	return h.Handler.Handle(ctx, r)
}

func (h DebugOnlyHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return DebugOnlyHandler{h.Handler.WithAttrs(attrs)}
}

func (h DebugOnlyHandler) WithGroup(name string) slog.Handler {
	return DebugOnlyHandler{h.Handler.WithGroup(name)}
}
