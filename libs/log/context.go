package log

import (
	"context"
	"log/slog"
)

type logger int

var loggerKey logger

type prefixKeyType struct{}

// WithPrefix returns a context that prepends prefix to every log message emitted from it.
// Calling WithPrefix on a context that already has a prefix appends with ": ".
func WithPrefix(ctx context.Context, prefix string) context.Context {
	if existing, _ := ctx.Value(prefixKeyType{}).(string); existing != "" {
		prefix = existing + ": " + prefix
	}
	return context.WithValue(ctx, prefixKeyType{}, prefix)
}

func getPrefix(ctx context.Context) string {
	prefix, _ := ctx.Value(prefixKeyType{}).(string)
	return prefix
}

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
