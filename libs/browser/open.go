package browser

import (
	"context"
	"io"
	"os"
	osexec "os/exec"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	browserpkg "github.com/pkg/browser"
)

const (
	browserEnvVar          = "BROWSER"
	disabledBrowser        = "none"
	defaultDisabledMessage = "Open this link in your browser:\n"
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

// shellName returns the shell executable name for the current OS.
func shellName() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	return "sh"
}

// shellFlag returns the flag to pass an inline command to the shell.
func shellFlag() string {
	if runtime.GOOS == "windows" {
		return "/c"
	}
	return "-c"
}

// containsQuotes reports whether s contains single or double quote characters.
func containsQuotes(s string) bool {
	return strings.ContainsAny(s, `"'`)
}

// parseBrowserCommand splits the BROWSER env var into a command slice.
// If the value contains quotes it delegates to the system shell so that
// values like `open -a "Google Chrome"` are handled correctly.
func parseBrowserCommand(raw string) []string {
	if raw == "" {
		return nil
	}
	if containsQuotes(raw) {
		return []string{shellName(), shellFlag(), raw}
	}
	return strings.Fields(raw)
}

// IsDisabled reports whether browser launching is disabled for the context.
func IsDisabled(ctx context.Context) bool {
	return env.Get(ctx, browserEnvVar) == disabledBrowser
}

// OpenerOption configures NewOpener.
type OpenerOption func(*openerConfig)

type openerConfig struct {
	disabledMessage string
}

// WithDisabledMessage overrides the message printed when BROWSER=none.
func WithDisabledMessage(msg string) OpenerOption {
	return func(cfg *openerConfig) {
		cfg.disabledMessage = msg
	}
}

// NewOpener returns a function that opens URLs in the browser.
func NewOpener(ctx context.Context, workingDirectory string, opts ...OpenerOption) func(string) error {
	cfg := &openerConfig{
		disabledMessage: defaultDisabledMessage,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	raw := env.Get(ctx, browserEnvVar)
	browserCommand := parseBrowserCommand(raw)
	switch {
	case len(browserCommand) == 0:
		return openDefaultBrowserURL
	case raw == disabledBrowser:
		return func(targetURL string) error {
			cmdio.LogString(ctx, cfg.disabledMessage+targetURL)
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
