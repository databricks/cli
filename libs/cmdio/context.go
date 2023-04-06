package cmdio

import (
	"context"

	"github.com/databricks/bricks/libs/flags"
)

type progressLogger int

var progressLoggerKey progressLogger

// NewContext returns a new Context that carries the specified progress logger.
func NewContext(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, progressLoggerKey, logger)
}

// FromContext returns the progress logger value stored in ctx, if any.
func FromContext(ctx context.Context) (*Logger, bool) {
	u, ok := ctx.Value(progressLoggerKey).(*Logger)
	return u, ok
}

// If current configured logger is inplace, then this function replaces it with
// append logger
func DisableInplace(ctx context.Context) context.Context {
	logger, ok := FromContext(ctx)
	if !ok {
		return ctx
	}
	if logger.Mode != flags.ModeInplace {
		return ctx
	}
	return NewContext(ctx, NewLogger(flags.ModeAppend))
}
