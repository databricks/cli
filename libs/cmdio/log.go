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
	c := fromContext(ctx)
	_, _ = io.WriteString(c.err, str+"\n")
}
