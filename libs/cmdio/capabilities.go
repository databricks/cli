package cmdio

import (
	"context"
	"io"
	"strings"

	"github.com/databricks/cli/libs/env"
)

// Capabilities represents terminal I/O capabilities detected from environment.
type Capabilities struct {
	// Raw TTY detection results
	stdinIsTTY  bool
	stdoutIsTTY bool
	stderrIsTTY bool

	// Environment flags
	color     bool // Color output is enabled (NO_COLOR not set and TERM not dumb)
	isGitBash bool // Git Bash on Windows
}

// newCapabilities detects terminal capabilities from context and I/O streams.
func newCapabilities(ctx context.Context, in io.Reader, out, err io.Writer) Capabilities {
	return Capabilities{
		stdinIsTTY:  isTTY(in),
		stdoutIsTTY: isTTY(out),
		stderrIsTTY: isTTY(err),
		color:       env.Get(ctx, "NO_COLOR") == "" && env.Get(ctx, "TERM") != "dumb",
		isGitBash:   detectGitBash(ctx),
	}
}

// SupportsInteractive returns true if terminal supports interactive features (colors, spinners).
func (c Capabilities) SupportsInteractive() bool {
	return c.stderrIsTTY && c.color
}

// SupportsPrompt returns true if terminal supports user prompting.
// Prompts write to stderr and read from stdin, so we only need those to be TTYs.
func (c Capabilities) SupportsPrompt() bool {
	return c.SupportsInteractive() && c.stdinIsTTY && !c.isGitBash
}

// SupportsColor returns true if the given writer supports colored output.
// This checks both TTY status and environment variables (NO_COLOR, TERM=dumb).
func (c Capabilities) SupportsColor(w io.Writer) bool {
	return isTTY(w) && c.color
}

// detectGitBash returns true if running in Git Bash on Windows (has broken promptui support).
// We do not allow prompting in Git Bash on Windows.
// Likely due to fact that Git Bash does not correctly support ANSI escape sequences,
// we cannot use promptui package there.
// See known issues:
// - https://github.com/manifoldco/promptui/issues/208
// - https://github.com/chzyer/readline/issues/191
func detectGitBash(ctx context.Context) bool {
	// Check if the MSYSTEM environment variable is set to "MINGW64"
	msystem := env.Get(ctx, "MSYSTEM")
	if strings.EqualFold(msystem, "MINGW64") {
		// Check for typical Git Bash env variable for prompts
		ps1 := env.Get(ctx, "PS1")
		return strings.Contains(ps1, "MINGW") || strings.Contains(ps1, "MSYSTEM")
	}

	return false
}
