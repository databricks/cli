package browser

import (
	"context"
	"fmt"
	"io"
	"os"
	osexec "os/exec"
	"runtime"

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

var runBrowserCommand = func(ctx context.Context, workingDirectory, browserRaw, targetURL string) error {
	// Always execute through the system shell. This is necessary on Windows
	// where scripts (.py, .bat, etc.) cannot be executed directly via exec.
	fullCmd := fmt.Sprintf("%s %q", browserRaw, targetURL)
	cmd := osexec.CommandContext(ctx, shellName(), shellFlag(), fullCmd)
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

// parseBrowserCommand returns the raw BROWSER value if set, or empty string
// if unset. The raw value is passed to the system shell for execution, so
// quoting and multi-word commands are handled by the shell.
func parseBrowserCommand(raw string) string {
	return raw
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
	switch browserCommand {
	case "":
		return openDefaultBrowserURL
	case disabledBrowser:
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
