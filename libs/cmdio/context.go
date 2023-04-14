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

// If the mode of the progress logger in ctx is default then this function
// tries to resolve it to inplace if it's supported, and otherwise to append
func TryResolveDefaultToInplace(ctx context.Context) {
	logger, ok := FromContext(ctx)
	if !ok {
		return
	}
	if logger.Mode != flags.ModeDefault {
		return
	}
	format := flags.ModeAppend
	if logger.isInplaceSupported {
		format = flags.ModeInplace
	}
	logger.Mode = format
}

// If the mode of the progress logger in ctx is default then this function resolves
// the mode to append
func ResolveDefaultToAppend(ctx context.Context) {
	logger, ok := FromContext(ctx)
	if !ok {
		return
	}
	if logger.Mode != flags.ModeDefault {
		return
	}
	logger.Mode = flags.ModeAppend
}
