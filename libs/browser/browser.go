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
	"io"
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
	return openURL(ctx, url, false)
}

// OpenOutputToStderr is Open for a caller whose stdout is a document, such as `--output json`. The output of the
// browser command goes to stderr, as its error output does. The other behaviors are Open's.
func OpenOutputToStderr(ctx context.Context, url string) error {
	return openURL(ctx, url, true)
}

// openURL implements Open and OpenOutputToStderr. stdoutToStderr sends the output of the browser to stderr.
func openURL(ctx context.Context, url string, stdoutToStderr bool) error {
	browserCmd := env.Get(ctx, "BROWSER")
	switch browserCmd {
	case "":
		return openDefault(url, stdoutToStderr)
	case "none":
		cmdio.LogString(ctx, "Open this URL in your browser:\n"+url)
		return nil
	default:
		return openWithCommand(ctx, browserCmd, url, stdoutToStderr)
	}
}

// IsDisabled reports whether BROWSER=none is set. Callers that want a custom
// message for this case should branch on IsDisabled and skip Open.
func IsDisabled(ctx context.Context) bool {
	return env.Get(ctx, "BROWSER") == "none"
}

func openDefault(url string, stdoutToStderr bool) error {
	// github.com/pkg/browser writes xdg-open's output to os.Stdout and its error output to os.Stderr, even when the
	// open succeeds, producing spurious noise on Linux desktops. The error output is discarded.
	originalStdout, originalStderr := browserpkg.Stdout, browserpkg.Stderr
	defer func() {
		browserpkg.Stdout = originalStdout
		browserpkg.Stderr = originalStderr
	}()
	browserpkg.Stderr = io.Discard
	if stdoutToStderr {
		browserpkg.Stdout = os.Stderr
	}
	return browserpkg.OpenURL(url)
}

func openWithCommand(ctx context.Context, browserCmd, url string, stdoutToStderr bool) error {
	e, err := exec.NewCommandExecutor(".")
	if err != nil {
		return err
	}
	if !stdoutToStderr {
		e.WithInheritOutput()
	}
	cmd, err := e.StartCommand(ctx, fmt.Sprintf("%q %q", browserCmd, url))
	if err != nil {
		return err
	}
	if !stdoutToStderr {
		return cmd.Wait()
	}
	// The command's stdout is a pipe, and Wait closes it, so it is copied to stderr before Wait.
	var copied sync.WaitGroup
	copied.Add(2)
	go func() {
		defer copied.Done()
		_, _ = io.Copy(os.Stderr, cmd.Stdout())
	}()
	go func() {
		defer copied.Done()
		_, _ = io.Copy(os.Stderr, cmd.Stderr())
	}()
	copied.Wait()
	return cmd.Wait()
}
