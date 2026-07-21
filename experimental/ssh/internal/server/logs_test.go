package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/libs/log"
	"github.com/stretchr/testify/assert"
)

func TestLogBufferEvictsOldestLines(t *testing.T) {
	buf := newLogBuffer(25)
	for _, line := range []string{"first line\n", "second line\n", "third line\n"} {
		_, err := buf.Write([]byte(line))
		assert.NoError(t, err)
	}
	out := buf.String()
	assert.NotContains(t, out, "first line")
	assert.Contains(t, out, "second line")
	assert.Contains(t, out, "third line")
}

func TestCaptureWarnLogsRecordsWarnAndAbove(t *testing.T) {
	ctx := log.NewContext(t.Context(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, buf := captureWarnLogs(ctx)

	log.Infof(ctx, "info message")
	log.Warnf(ctx, "warn message")
	log.Errorf(ctx, "error message")

	out := buf.String()
	assert.NotContains(t, out, "info message")
	assert.Contains(t, out, "warn message")
	assert.Contains(t, out, "error message")
}

func TestCaptureWarnLogsKeepsHandlerAttrs(t *testing.T) {
	ctx := log.NewContext(t.Context(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, buf := captureWarnLogs(ctx)
	// Mirrors how the proxy server scopes its logger per connection.
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("session", "abc"))

	log.Errorf(ctx, "connection failed")

	out := buf.String()
	assert.Contains(t, out, "connection failed")
	assert.Contains(t, out, "session=abc")
}

func TestLogBufferServeHTTP(t *testing.T) {
	ctx := log.NewContext(t.Context(), slog.New(slog.NewTextHandler(io.Discard, nil)))
	ctx, buf := captureWarnLogs(ctx)
	log.Errorf(ctx, "boom")

	rec := httptest.NewRecorder()
	buf.serveHTTP(rec, httptest.NewRequest(http.MethodGet, "/logs", nil))

	assert.Equal(t, 200, rec.Code)
	assert.Equal(t, "text/plain; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), "boom")
}
