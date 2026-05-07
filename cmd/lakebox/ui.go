package lakebox

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

// cmdioSpinner is the subset of *cmdio.spinner's method set we need.
// Defining the interface locally lets us hold the unexported type as a
// struct field; cmdio's spinner satisfies it structurally.
type cmdioSpinner interface {
	Close()
}

// spinner wraps cmdio.NewSpinner with ok/fail markers. ok and fail close the
// underlying spinner and log a final ✓/✗ line; Close stops the spinner
// without printing. cmdio's Close is itself idempotent, so a `defer s.Close()`
// is safe alongside an ok/fail call on the success path.
type spinner struct {
	cmdioSpinner
	ctx context.Context
}

func spin(ctx context.Context, msg string) *spinner {
	sp := cmdio.NewSpinner(ctx)
	sp.Update(msg)
	return &spinner{cmdioSpinner: sp, ctx: ctx}
}

func (s *spinner) ok(msg string)   { s.mark("✓", msg) }
func (s *spinner) fail(msg string) { s.mark("✗", msg) }

func (s *spinner) mark(mark, msg string) {
	s.Close()
	cmdio.LogString(s.ctx, "  "+cmdio.Cyan(s.ctx, mark)+" "+msg)
}

// status formats a lakebox lifecycle status with a color hint.
func status(ctx context.Context, s string) string {
	switch strings.ToLower(s) {
	case "running":
		return cmdio.Cyan(ctx, "running")
	case "stopped":
		return cmdio.Dim(ctx, "stopped")
	case "creating":
		return cmdio.Bold(ctx, cmdio.Cyan(ctx, "creating…"))
	default:
		return cmdio.Dim(ctx, strings.ToLower(s))
	}
}

// field prints "  label  value" to w, where label is dimmed and padded to a
// fixed visible width. Padding has to happen before Dim so the SGR escapes
// don't inflate the byte count and break column alignment.
func field(ctx context.Context, w io.Writer, label, value string) {
	fmt.Fprintf(w, "  %s %s\n", cmdio.Dim(ctx, fmt.Sprintf("%-10s", label)), value)
}

// ok prints "  ✓ message" to stderr via the cmdio context.
func ok(ctx context.Context, msg string) {
	cmdio.LogString(ctx, "  "+cmdio.Cyan(ctx, "✓")+" "+msg)
}

// warn prints "  ! message" to stderr via the cmdio context.
func warn(ctx context.Context, msg string) {
	cmdio.LogString(ctx, "  "+cmdio.Cyan(ctx, "!")+" "+msg)
}

// blank prints an empty line to w.
func blank(w io.Writer) {
	fmt.Fprintln(w)
}
