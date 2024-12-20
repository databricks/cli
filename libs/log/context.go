package log

import (
	"context"
	"log/slog"
)

type logger int

var loggerKey logger

// NewContext returns a new Context that carries the specified logger.
//
// Discussion why this is not part of slog itself: https://github.com/golang/go/issues/58243.
func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the Logger value stored in ctx, if any.
//
// Discussion why this is not part of slog itself: https://github.com/golang/go/issues/58243.
func FromContext(ctx context.Context) (*slog.Logger, bool) {
	u, ok := ctx.Value(loggerKey).(*slog.Logger)
	return u, ok
}
