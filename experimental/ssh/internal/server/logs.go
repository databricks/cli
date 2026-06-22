package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/log"
)

// maxLogBufferBytes bounds the in-memory log buffer served at /logs.
const maxLogBufferBytes = 16 * 1024

// logBuffer accumulates recent log records in memory so the client can fetch them
// via the /logs endpoint after a failed connection attempt. The Jobs API exposes no
// stdout logs for a running notebook task, so this is the only way for "ssh connect"
// to read the server's errors while the bootstrap job is still alive.
type logBuffer struct {
	mu    sync.Mutex
	lines []string
	size  int
	limit int
}

func newLogBuffer(limit int) *logBuffer {
	return &logBuffer{limit: limit}
}

// Write implements io.Writer; slog handlers emit one Write call per record.
func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.lines = append(b.lines, string(p))
	b.size += len(p)
	for b.size > b.limit && len(b.lines) > 0 {
		b.size -= len(b.lines[0])
		b.lines = b.lines[1:]
	}
	return len(p), nil
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return strings.Join(b.lines, "")
}

func (b *logBuffer) serveHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if _, err := io.WriteString(w, b.String()); err != nil {
		http.Error(w, "Failed to write logs", http.StatusInternalServerError)
	}
}

// captureWarnLogs returns a context whose logger also records warning-and-above
// log lines into the returned buffer.
func captureWarnLogs(ctx context.Context) (context.Context, *logBuffer) {
	buf := newLogBuffer(maxLogBufferBytes)
	bufHandler := slog.NewTextHandler(buf, &slog.HandlerOptions{Level: slog.LevelWarn})
	logger := slog.New(teeHandler{[]slog.Handler{log.GetLogger(ctx).Handler(), bufHandler}})
	return log.NewContext(ctx, logger), buf
}

// teeHandler forwards log records to all underlying handlers.
type teeHandler struct {
	handlers []slog.Handler
}

func (t teeHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, h := range t.handlers {
		if h.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (t teeHandler) Handle(ctx context.Context, r slog.Record) error {
	var errs []error
	for _, h := range t.handlers {
		if h.Enabled(ctx, r.Level) {
			errs = append(errs, h.Handle(ctx, r.Clone()))
		}
	}
	return errors.Join(errs...)
}

func (t teeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		handlers[i] = h.WithAttrs(attrs)
	}
	return teeHandler{handlers}
}

func (t teeHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(t.handlers))
	for i, h := range t.handlers {
		handlers[i] = h.WithGroup(name)
	}
	return teeHandler{handlers}
}
