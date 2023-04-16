package cmdio

import (
	"context"
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
