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
	browserCmd := env.Get(ctx, "BROWSER")
	switch browserCmd {
	case "":
		return openDefault(url)
	case "none":
		cmdio.LogString(ctx, "Open this URL in your browser:\n"+url)
		return nil
	default:
		return openWithCommand(ctx, browserCmd, url)
	}
}

// IsDisabled reports whether BROWSER=none is set. Callers that want a custom
// message for this case should branch on IsDisabled and skip Open.
func IsDisabled(ctx context.Context) bool {
	return env.Get(ctx, "BROWSER") == "none"
}

func openDefault(url string) error {
	// github.com/pkg/browser writes xdg-open's error output to os.Stderr even
	// when the open succeeds, producing spurious noise on Linux desktops.
	originalStderr := browserpkg.Stderr
	defer func() {
		browserpkg.Stderr = originalStderr
	}()
	browserpkg.Stderr = io.Discard
	return browserpkg.OpenURL(url)
}

func openWithCommand(ctx context.Context, browserCmd, url string) error {
	e, err := exec.NewCommandExecutor(".")
	if err != nil {
		return err
	}
	e.WithInheritOutput()
	cmd, err := e.StartCommand(ctx, fmt.Sprintf("%q %q", browserCmd, url))
	if err != nil {
		return err
	}
	return cmd.Wait()
}
