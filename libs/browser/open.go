package browser

import (
	"context"
	"io"
	"os"
	osexec "os/exec"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	browserpkg "github.com/pkg/browser"
)

const (
	browserEnvVar   = "BROWSER"
	disabledBrowser = "none"
)

var openDefaultBrowserURL = func(targetURL string) error {
	originalStderr := browserpkg.Stderr
	defer func() {
		browserpkg.Stderr = originalStderr
	}()

	browserpkg.Stderr = io.Discard
	return browserpkg.OpenURL(targetURL)
}

var runBrowserCommand = func(ctx context.Context, workingDirectory string, browserCommand []string, targetURL string) error {
	cmd := osexec.CommandContext(ctx, browserCommand[0], append(browserCommand[1:], targetURL)...)
	cmd.Dir = workingDirectory
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// IsDisabled reports whether browser launching is disabled for the context.
func IsDisabled(ctx context.Context) bool {
	return env.Get(ctx, browserEnvVar) == disabledBrowser
}

// NewOpener returns a function that opens URLs in the browser.
func NewOpener(ctx context.Context, workingDirectory string) func(string) error {
	browserCommand := strings.Fields(env.Get(ctx, browserEnvVar))
	switch {
	case len(browserCommand) == 0:
		return openDefaultBrowserURL
	case len(browserCommand) == 1 && browserCommand[0] == disabledBrowser:
		return func(targetURL string) error {
			cmdio.LogString(ctx, "Please complete authentication by opening this link in your browser:\n"+targetURL)
			return nil
		}
	default:
		return func(targetURL string) error {
			return runBrowserCommand(ctx, workingDirectory, browserCommand, targetURL)
		}
	}
}

// OpenURL opens a URL in the browser.
func OpenURL(ctx context.Context, workingDirectory, targetURL string) error {
	return NewOpener(ctx, workingDirectory)(targetURL)
}
