package logger

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"golang.org/x/exp/slog"
)

// Get returns either the logger configured on the context,
// or the global logger if one isn't defined.
func Get(ctx context.Context) *slog.Logger {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = slog.Default()
	}
	return logger
}

// helper function to abstract logging a string message.
func log(logger *slog.Logger, ctx context.Context, level slog.Level, msg string) {
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller].
	runtime.Callers(3, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	if ctx == nil {
		ctx = context.Background()
	}
	_ = logger.Handler().Handle(ctx, r)
}

// Tracef logs a formatted string using the context-local or global logger.
func Tracef(ctx context.Context, format string, v ...any) {
	logger := Get(ctx)
	if !logger.Enabled(ctx, LevelTrace) {
		return
	}
	log(logger, ctx, LevelTrace, fmt.Sprintf(format, v...))
}

// Debugf logs a formatted string using the context-local or global logger.
func Debugf(ctx context.Context, format string, v ...any) {
	logger := Get(ctx)
	if !logger.Enabled(ctx, LevelDebug) {
		return
	}
	log(logger, ctx, LevelDebug, fmt.Sprintf(format, v...))

}

// Infof logs a formatted string using the context-local or global logger.
func Infof(ctx context.Context, format string, v ...any) {
	logger := Get(ctx)
	if !logger.Enabled(ctx, LevelInfo) {
		return
	}
	log(logger, ctx, LevelInfo, fmt.Sprintf(format, v...))
}

// Warnf logs a formatted string using the context-local or global logger.
func Warnf(ctx context.Context, format string, v ...any) {
	logger := Get(ctx)
	if !logger.Enabled(ctx, LevelWarn) {
		return
	}
	log(logger, ctx, LevelWarn, fmt.Sprintf(format, v...))
}

// Errorf logs a formatted string using the context-local or global logger.
func Errorf(ctx context.Context, format string, v ...any) {
	logger := Get(ctx)
	if !logger.Enabled(ctx, LevelError) {
		return
	}
	log(logger, ctx, LevelError, fmt.Sprintf(format, v...))
}
