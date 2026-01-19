package cmdio

import (
	"io"
	"os"

	"github.com/mattn/go-isatty"
)

// isTTY detects if the given reader or writer is a terminal.
func isTTY(v any) bool {
	// Check if it's a fakeTTY first.
	if _, ok := v.(*fakeTTY); ok {
		return true
	}

	// Check if it's an actual TTY.
	f, ok := v.(*os.File)
	if !ok {
		return false
	}
	fd := f.Fd()
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}

// fakeTTY wraps an io.Writer and makes IsTTY return true for it.
// This is useful for testing TTY-dependent behavior without requiring an actual terminal.
type fakeTTY struct {
	io.Writer
}

// FakeTTY creates a writer that IsTTY will recognize as a TTY.
// This is useful for testing terminal-dependent behavior.
func FakeTTY(w io.Writer) io.Writer {
	return &fakeTTY{Writer: w}
}
