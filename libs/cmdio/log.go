package cmdio

import (
	"context"
	"fmt"
	"io"

	"github.com/databricks/cli/libs/textutil"
)

// Log calls [LogString] with the string representation of str.
func Log(ctx context.Context, str fmt.Stringer) {
	LogString(ctx, str.String())
}

// LogString writes str to the error writer, followed by a newline. str is
// usually the CLI's own text around a value from an API response (a name, a
// state message), so terminal control characters in it are escaped; the colors
// the CLI added are kept.
func LogString(ctx context.Context, str string) {
	c := fromContext(ctx)
	_, _ = io.WriteString(c.err, textutil.EscapeControlKeepColors(str)+"\n")
}

// LogRawString writes str to the error writer as is, followed by a newline. It
// is for output passed through from a local process, such as a bundle script,
// where escape sequences (a progress bar, cursor movement) are the process's
// own. Everything else goes through [LogString].
func LogRawString(ctx context.Context, str string) {
	c := fromContext(ctx)
	_, _ = io.WriteString(c.err, str+"\n")
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
