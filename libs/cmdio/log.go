package cmdio

import (
	"context"
	"fmt"
	"io"
)

// Log calls [LogString] with the string representation of str.
func Log(ctx context.Context, str fmt.Stringer) {
	LogString(ctx, str.String())
}

// LogString writes str to the error writer, followed by a newline.
func LogString(ctx context.Context, str string) {
	// Callers style str with cmdio.Yellow and friends, which check stdout's
	// color support, but str goes to stderr. With stdout a terminal and stderr
	// redirected, the escapes are stripped here.
	_, _ = io.WriteString(fromContext(ctx).stderrWriter(ctx), str+"\n")
}

type quietKey struct{}

// WithQuiet returns a context in which [LogProgress] writes nothing. Only
// progress output is affected: diagnostics go through libs/logdiag, and command
// results go through [LogString], so neither is suppressed.
func WithQuiet(ctx context.Context) context.Context {
	return context.WithValue(ctx, quietKey{}, true)
}

// IsQuiet reports whether progress output is suppressed on this context.
func IsQuiet(ctx context.Context) bool {
	quiet, ok := ctx.Value(quietKey{}).(bool)
	return ok && quiet
}

// LogProgress writes str like [LogString], unless the context is marked quiet by
// [WithQuiet]. Use it for messages that only report what the command is doing
// ("Uploading ...", "Building ..."), not for results the user asked for.
func LogProgress(ctx context.Context, str string) {
	if IsQuiet(ctx) {
		return
	}
	LogString(ctx, str)
}
