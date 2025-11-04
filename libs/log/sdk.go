package log

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"

	sdk "github.com/databricks/databricks-sdk-go/logger"
)

// slogAdapter makes an slog.Logger usable with the Databricks SDK.
type slogAdapter struct{}

func (s slogAdapter) Enabled(ctx context.Context, level sdk.Level) bool {
	logger := GetLogger(ctx)
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

func (s slogAdapter) log(ctx context.Context, logger *slog.Logger, level slog.Level, msg string) {
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
	logger := GetLogger(ctx)
	if !logger.Enabled(ctx, LevelTrace) {
		return
	}
	s.log(ctx, logger, LevelTrace, fmt.Sprintf(format, v...))
}

func (s slogAdapter) Debugf(ctx context.Context, format string, v ...any) {
	logger := GetLogger(ctx)
	if !logger.Enabled(ctx, LevelDebug) {
		return
	}
	s.log(ctx, logger, LevelDebug, fmt.Sprintf(format, v...))
}

func (s slogAdapter) Infof(ctx context.Context, format string, v ...any) {
	logger := GetLogger(ctx)
	if !logger.Enabled(ctx, LevelInfo) {
		return
	}
	s.log(ctx, logger, LevelInfo, fmt.Sprintf(format, v...))
}

func (s slogAdapter) Warnf(ctx context.Context, format string, v ...any) {
	logger := GetLogger(ctx)
	if !logger.Enabled(ctx, LevelWarn) {
		return
	}
	s.log(ctx, logger, LevelWarn, fmt.Sprintf(format, v...))
}

func (s slogAdapter) Errorf(ctx context.Context, format string, v ...any) {
	logger := GetLogger(ctx)
	if !logger.Enabled(ctx, LevelError) {
		return
	}
	s.log(ctx, logger, LevelError, fmt.Sprintf(format, v...))
}

func init() {
	// Configure SDK to use this logger.
	sdk.DefaultLogger = slogAdapter{}
}
