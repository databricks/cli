package cmdio

import (
	"context"
	"io"

	"github.com/databricks/cli/libs/env"
)

// Capabilities represents terminal I/O capabilities detected from environment.
type Capabilities struct {
	// Raw TTY detection results
	stdinIsTTY  bool
	stdoutIsTTY bool
	stderrIsTTY bool

	// stdinIsMSYSPipe is true when stdin is an MSYS2/Cygwin pty pipe rather
	// than a real console (no ConPTY; see conpty.go). bubbletea can't read
	// keystrokes from it, so prompts and pagers are disabled there.
	stdinIsMSYSPipe bool

	// Environment flags. They are independent on purpose: NO_COLOR is a
	// color preference only (https://no-color.org), while TERM=dumb means
	// the terminal can't move the cursor, which rules out redrawing output.
	noColor bool // NO_COLOR is set
	dumb    bool // TERM=dumb
}

// newCapabilities detects terminal capabilities from context and I/O streams.
func newCapabilities(ctx context.Context, in io.Reader, out, err io.Writer) Capabilities {
	return Capabilities{
		stdinIsTTY:      isTTY(in),
		stdoutIsTTY:     isTTY(out),
		stderrIsTTY:     isTTY(err),
		noColor:         env.Get(ctx, "NO_COLOR") != "",
		dumb:            env.Get(ctx, "TERM") == "dumb",
		stdinIsMSYSPipe: isMSYSPipe(in),
	}
}

// SupportsInteractive returns true if stderr can show output that redraws
// in place (spinners, prompts). NO_COLOR does not affect this; it only
// removes color from that output.
func (c Capabilities) SupportsInteractive() bool {
	return c.stderrIsTTY && !c.dumb
}

// colorAllowed reports whether the environment permits color at all.
func (c Capabilities) colorAllowed() bool {
	return !c.noColor && !c.dumb
}

// SupportsPrompt returns true if terminal supports user prompting.
// Prompts write to stderr and read keystrokes from stdin, so stderr must be a
// TTY and stdin a console rather than an MSYS2 pty pipe (see conpty.go).
func (c Capabilities) SupportsPrompt() bool {
	return c.SupportsInteractive() && c.stdinIsTTY && !c.stdinIsMSYSPipe
}

// SupportsLinePrompt returns true if the terminal supports a plain-text line
// prompt (see [Ask]): it writes the question to stderr and reads the answer
// from stdin. stdout must be a terminal too: when it is piped to a pager, the
// pager reads the same terminal, so its keystrokes could answer the prompt.
// Unlike SupportsPrompt it does not depend on TERM=dumb or an MSYS2 pty pipe
// (see conpty.go), which only matter for the TUI prompts.
func (c Capabilities) SupportsLinePrompt() bool {
	return c.stdinIsTTY && c.stdoutIsTTY && c.stderrIsTTY
}

// SupportsColor returns true if the given writer supports colored output.
// This checks both TTY status and environment variables (NO_COLOR, TERM=dumb).
func (c Capabilities) SupportsColor(w io.Writer) bool {
	return isTTY(w) && c.colorAllowed()
}

// SupportsStdoutColor returns true if stdout supports colored output.
// Use this when emitting colored bytes to a writer that wraps stdout (e.g.
// a buffered flusher) where SupportsColor's isTTY check would be misled.
func (c Capabilities) SupportsStdoutColor() bool {
	return c.stdoutIsTTY && c.colorAllowed()
}

// SupportsStderrColor is the stderr counterpart of SupportsStdoutColor.
// Prompts use it; they render to stderr.
func (c Capabilities) SupportsStderrColor() bool {
	return c.stderrIsTTY && c.colorAllowed()
}

// SupportsPager returns true when we can drive an interactive pager.
// It builds on SupportsPrompt (stderr+stdin TTY, stdin not an MSYS2 pipe) and
// additionally requires stdout to be a TTY so rendered rows land on
// the terminal rather than a redirected file.
func (c Capabilities) SupportsPager() bool {
	return c.SupportsPrompt() && c.stdoutIsTTY
}

// InteractiveMode represents the level of terminal interactivity available.
type InteractiveMode string

// Interactive mode constants for user agent tracking.
const (
	InteractiveModeFull       InteractiveMode = "full"   // Both interactive output and prompts supported
	InteractiveModeOutputOnly InteractiveMode = "output" // Interactive output only, no prompts (stdin not a TTY, or an MSYS2 pipe)
	InteractiveModeNone       InteractiveMode = "none"   // Non-interactive (CI, cron, stderr redirected)
)

// InteractiveMode returns the interactive mode based on terminal capabilities.
func (c Capabilities) InteractiveMode() InteractiveMode {
	// SupportsPrompt() implies SupportsInteractive() (it's a stricter check).
	if c.SupportsPrompt() {
		return InteractiveModeFull
	}
	if c.SupportsInteractive() {
		return InteractiveModeOutputOnly
	}
	return InteractiveModeNone
}
