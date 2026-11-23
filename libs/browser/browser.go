// Package browser opens URLs in the user's browser, respecting the BROWSER
// environment variable.
//
// The {empty, none, <command>} semantics match common conventions (xdg-open,
// GitHub CLI, etc.). The <command> path runs through libs/exec to preserve
// Windows shell escaping for percent-encoded URLs; a prior inline "cmd /c"
// implementation corrupted OAuth redirect URLs on Windows and was reverted.
package browser

import (
	"context"
	"fmt"
	"os"
	"sync"

	browserpkg "github.com/pkg/browser"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/exec"
)

// Open launches url in the user's browser.
//
// Behavior by BROWSER environment variable:
//
//   - unset: opens the default system browser
//   - "none": prints url to stderr and returns nil without opening anything
//   - any other value: runs it as a command with url as the single argument
//
// Callers that want to override the "none" message should branch on IsDisabled
// before calling Open.
func Open(ctx context.Context, url string) error {
	return openURL(ctx, url, false, false)
}

// OpenOutputToStderr is Open for a caller whose stdout is a document, such as `--output json`. The output of the
// browser command goes to stderr, as its error output does. The other behaviors are Open's.
func OpenOutputToStderr(ctx context.Context, url string) error {
	return openURL(ctx, url, true, false)
}

// OpenOutputToStderrWithoutStdin is OpenOutputToStderr for a caller whose stdin is a protocol stream, such as the stdin of
// an MCP server. A browser command gets no stdin, so it cannot read that stream. The other behaviors are OpenOutputToStderr's.
func OpenOutputToStderrWithoutStdin(ctx context.Context, url string) error {
	return openURL(ctx, url, true, true)
}

// openURL implements the exported openers. stdoutToStderr sends the output of the browser to stderr, and noStdin gives a
// browser command no stdin. The default opener has no stdin to give up.
func openURL(ctx context.Context, url string, stdoutToStderr, noStdin bool) error {
	browserCmd := env.Get(ctx, "BROWSER")
	switch browserCmd {
	case "":
		return openDefault(url, stdoutToStderr)
	case "none":
		cmdio.LogString(ctx, "Open this URL in your browser:\n"+url)
		return nil
	default:
		return openWithCommand(ctx, browserCmd, url, stdoutToStderr, noStdin)
	}
}

// IsDisabled reports whether BROWSER=none is set. Callers that want a custom
// message for this case should branch on IsDisabled and skip Open.
func IsDisabled(ctx context.Context) bool {
	return env.Get(ctx, "BROWSER") == "none"
}

// defaultOpenMu serializes the default opener, which swaps the streams of pkg/browser for the duration of an open.
// Sign-in can start an open while another one runs.
var defaultOpenMu sync.Mutex

// openDefaultURL is the launcher of the default opener. Tests replace it, so that no browser opens on any OS.
var openDefaultURL = browserpkg.OpenURL

// openDefault opens url with pkg/browser. stdoutToStderr sends the output of the launcher to stderr, as
// OpenOutputToStderr does. The launcher's error output is discarded.
func openDefault(url string, stdoutToStderr bool) error {
	// github.com/pkg/browser writes xdg-open's output to os.Stdout and its error output to os.Stderr, even when the
	// open succeeds, producing spurious noise on Linux desktops. The error output is discarded. The streams are the
	// package variables Stdout and Stderr (https://github.com/pkg/browser/blob/5ac0b6a4141c/browser.go).
	//
	// The streams must be files. exec passes a file to the command as it is, so the command's Run returns when the
	// launcher exits. Any other writer is copied through a pipe, and Run waits for the pipe to close, which a browser
	// started by the launcher keeps open.
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer devNull.Close()

	defaultOpenMu.Lock()
	defer defaultOpenMu.Unlock()
	originalStdout, originalStderr := browserpkg.Stdout, browserpkg.Stderr
	defer func() {
		browserpkg.Stdout = originalStdout
		browserpkg.Stderr = originalStderr
	}()
	browserpkg.Stderr = devNull
	if stdoutToStderr {
		browserpkg.Stdout = os.Stderr
	}
	return openDefaultURL(url)
}

// openWithCommand runs browserCmd with url as its argument, through libs/exec, and waits for it to exit. Its output goes
// to stdout, or to stderr when stdoutToStderr is set. It reads no stdin when noStdin is set.
func openWithCommand(ctx context.Context, browserCmd, url string, stdoutToStderr, noStdin bool) error {
	e, err := exec.NewCommandExecutor(".")
	if err != nil {
		return err
	}
	if stdoutToStderr {
		e.WithInheritOutputToStderr()
	} else {
		e.WithInheritOutput()
	}
	if noStdin {
		e.WithoutStdin()
	}
	cmd, err := e.StartCommand(ctx, fmt.Sprintf("%q %q", browserCmd, url))
	if err != nil {
		return err
	}
	return cmd.Wait()
}
