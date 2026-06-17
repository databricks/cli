package sandbox

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// jsonOutput reports whether the user asked for JSON output, via either
// the local `--json` flag or the framework-global `-o json`.
func jsonOutput(cmd *cobra.Command, jsonFlag bool) bool {
	if jsonFlag {
		return true
	}
	if f := cmd.Flag("output"); f != nil && f.Value.String() == "json" {
		return true
	}
	return false
}

// cmdioSpinner is the subset of *cmdio.spinner's method set we need,
// defined locally so we can hold the unexported type as a field.
type cmdioSpinner interface {
	Update(msg string)
	Close()
}

// spinner wraps cmdio.NewSpinner with ok/fail markers. Close (idempotent
// in cmdio) is safe alongside ok/fail, so callers can `defer s.Close()`.
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

// status formats a sandbox lifecycle status with a color hint.
func status(ctx context.Context, s string) string {
	switch strings.ToLower(s) {
	case "running":
		return cmdio.Cyan(ctx, "running")
	case "stopped":
		return cmdio.Faint(ctx, "stopped")
	case "creating":
		return cmdio.Bold(ctx, cmdio.Cyan(ctx, "creating…"))
	case "stopping":
		return cmdio.Yellow(ctx, "stopping…")
	default:
		return cmdio.Faint(ctx, strings.ToLower(s))
	}
}

// field prints "  label  value" to w, where label is dimmed and padded to a
// fixed visible width. Padding has to happen before Dim so the SGR escapes
// don't inflate the byte count and break column alignment.
func field(ctx context.Context, w io.Writer, label, value string) {
	fmt.Fprintf(w, "  %s %s\n", cmdio.Faint(ctx, fmt.Sprintf("%-10s", label)), value)
}

// ok prints "  ✓ message" to stderr via the cmdio context.
func ok(ctx context.Context, msg string) {
	cmdio.LogString(ctx, "  "+cmdio.Cyan(ctx, "✓")+" "+msg)
}

// warn prints "  ! message" to stderr via the cmdio context. Yellow so
// it visually differs from `ok`'s cyan ✓ and `spinner` cyan markers.
func warn(ctx context.Context, msg string) {
	cmdio.LogString(ctx, "  "+cmdio.Yellow(ctx, "!")+" "+msg)
}

// blank prints an empty line to w.
func blank(w io.Writer) {
	fmt.Fprintln(w)
}
