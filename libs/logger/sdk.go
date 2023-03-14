package logger

import (
	"context"
	"fmt"
	"runtime"
	"time"

	sdk "github.com/databricks/databricks-sdk-go/logger"
	"golang.org/x/exp/slog"
)

// slogAdapter makes an slog.Logger usable with the Databricks SDK.
type slogAdapter struct{}

func (s slogAdapter) Enabled(ctx context.Context, level sdk.Level) bool {
	logger := Get(ctx)
	switch level {
	case sdk.LevelTrace:
		return logger.Enabled(ctx, LevelTrace)
	case sdk.LevelDebug:
		return logger.Enabled(ctx, LevelDebug)
	case sdk.LevelInfo:
		return logger.Enabled(ctx, LevelInfo)
	case sdk.LevelWarn:
		return logger.Enabled(ctx, LevelWarn)
	case sdk.LevelError:
		return logger.Enabled(ctx, LevelError)
	default:
		return true
	}
}

func (s slogAdapter) log(logger *slog.Logger, ctx context.Context, level slog.Level, msg string) {
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller, the caller in the SDK].
	runtime.Callers(4, pcs[:])
	r := slog.NewRecord(time.Now(), level, msg, pcs[0])
	r.AddAttrs(slog.Bool("sdk", true))
	if ctx == nil {
		ctx = context.Background()
	}
	_ = logger.Handler().Handle(ctx, r)
}

func (s slogAdapter) Tracef(ctx context.Context, format string, v ...any) {
	logger := Get(ctx)
	if !logger.Enabled(ctx, LevelTrace) {
		return
	}
	s.log(logger, ctx, LevelTrace, fmt.Sprintf(format, v...))
}

func (s slogAdapter) Debugf(ctx context.Context, format string, v ...any) {
	logger := Get(ctx)
	if !logger.Enabled(ctx, LevelDebug) {
		return
	}
	s.log(logger, ctx, LevelDebug, fmt.Sprintf(format, v...))
}

func (s slogAdapter) Infof(ctx context.Context, format string, v ...any) {
	logger := Get(ctx)
	if !logger.Enabled(ctx, LevelInfo) {
		return
	}
	s.log(logger, ctx, LevelInfo, fmt.Sprintf(format, v...))
}

func (s slogAdapter) Warnf(ctx context.Context, format string, v ...any) {
	logger := Get(ctx)
	if !logger.Enabled(ctx, LevelWarn) {
		return
	}
	s.log(logger, ctx, LevelWarn, fmt.Sprintf(format, v...))
}

func (s slogAdapter) Errorf(ctx context.Context, format string, v ...any) {
	logger := Get(ctx)
	if !logger.Enabled(ctx, LevelError) {
		return
	}
	s.log(logger, ctx, LevelError, fmt.Sprintf(format, v...))
}

func init() {
	// Configure SDK to use this logger.
	sdk.DefaultLogger = slogAdapter{}
}
