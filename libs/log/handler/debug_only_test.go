package handler

import (
	"bytes"
	"log/slog"
	"testing"

	"github.com/databricks/cli/libs/log"
	"github.com/stretchr/testify/assert"
)

func TestDebugOnlyHandlerLogsWarningsAtDebug(t *testing.T) {
	var buf bytes.Buffer
	base := slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: log.LevelDebug})
	logger := slog.New(DebugOnlyHandler{Handler: base}.WithAttrs([]slog.Attr{slog.String("k", "v")}).(DebugOnlyHandler).WithGroup("g"))

	logger.Warn("failed to fetch")
	assert.Contains(t, buf.String(), "level=DEBUG")
	assert.NotContains(t, buf.String(), "level=WARN")
}

func TestDebugOnlyHandlerEnabledClampsLevel(t *testing.T) {
	debugBase := slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: log.LevelDebug})
	assert.True(t, DebugOnlyHandler{Handler: debugBase}.Enabled(t.Context(), slog.LevelWarn))

	// A warning is still filtered when the base handler's level is above Debug.
	errorBase := slog.NewTextHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelError})
	assert.False(t, DebugOnlyHandler{Handler: errorBase}.Enabled(t.Context(), slog.LevelWarn))
}
