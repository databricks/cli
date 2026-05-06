package lakebox

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

// spinner wraps cmdio.NewSpinner with terminal ok/fail markers. After the
// first call to ok or fail, the spinner is closed and a final line is logged
// to stderr; subsequent calls are no-ops.
type spinner struct {
	ctx      context.Context
	close    func()
	finished bool
}

func spin(ctx context.Context, msg string) *spinner {
	sp := cmdio.NewSpinner(ctx)
	sp.Update(msg)
	return &spinner{ctx: ctx, close: sp.Close}
}

func (s *spinner) ok(msg string)   { s.done("✓", msg) }
func (s *spinner) fail(msg string) { s.done("✗", msg) }

func (s *spinner) done(mark, msg string) {
	if s.finished {
		return
	}
	s.finished = true
	s.close()
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
