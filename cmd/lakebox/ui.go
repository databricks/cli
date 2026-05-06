package lakebox

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
)

// ANSI escapes for inline highlighting. cmdio handles terminal capability
// detection for the spinner, so we don't gate these on TTY here — strings
// piped to a non-terminal still carry the codes, matching the behavior of
// other CLI commands that call bold/dim helpers.
const (
	rs   = "\033[0m"  // reset
	bo   = "\033[1m"  // bold
	dm   = "\033[2m"  // dim
	cyan = "\033[36m" // accent
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
	cmdio.LogString(s.ctx, "  "+cyan+mark+rs+" "+msg)
}

// status formats a lakebox lifecycle status with the accent color.
func status(s string) string {
	switch strings.ToLower(s) {
	case "running":
		return cyan + "running" + rs
	case "stopped":
		return dm + "stopped" + rs
	case "creating":
		return cyan + bo + "creating…" + rs
	default:
		return dm + strings.ToLower(s) + rs
	}
}

// field prints "  label  value" to w.
func field(w io.Writer, label, value string) {
	fmt.Fprintf(w, "  %s%-10s%s %s\n", dm, label, rs, value)
}

// ok prints "  ✓ message" to stderr via the cmdio context.
func ok(ctx context.Context, msg string) {
	cmdio.LogString(ctx, "  "+cyan+"✓"+rs+" "+msg)
}

// warn prints "  ! message" to stderr via the cmdio context.
func warn(ctx context.Context, msg string) {
	cmdio.LogString(ctx, "  "+cyan+"!"+rs+" "+msg)
}

// blank prints an empty line to w.
func blank(w io.Writer) {
	fmt.Fprintln(w)
}

func accent(s string) string { return cyan + s + rs }
func bold(s string) string   { return bo + s + rs }
func dim(s string) string    { return dm + s + rs }
