package browser

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenURLUsesDefaultBrowser(t *testing.T) {
	original := openDefaultBrowserURL
	t.Cleanup(func() {
		openDefaultBrowserURL = original
	})

	var got string
	openDefaultBrowserURL = func(targetURL string) error {
		got = targetURL
		return nil
	}

	err := OpenURL(t.Context(), ".", "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com", got)
}

func TestOpenURLWithDisabledBrowser(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.Set(ctx, browserEnvVar, disabledBrowser)

	err := OpenURL(ctx, ".", "https://example.com")
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Open this link in your browser:")
	assert.Contains(t, stderr.String(), "https://example.com")
}

func TestOpenURLWithDisabledBrowserCustomMessage(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.Set(ctx, browserEnvVar, disabledBrowser)

	opener := NewOpener(ctx, ".", WithDisabledMessage("Custom message:\n"))
	err := opener("https://example.com")
	require.NoError(t, err)
	assert.Contains(t, stderr.String(), "Custom message:\nhttps://example.com")
}

func TestOpenURLUsesCustomBrowserCommand(t *testing.T) {
	original := runBrowserCommand
	t.Cleanup(func() {
		runBrowserCommand = original
	})

	ctx := env.Set(t.Context(), browserEnvVar, "custom-browser --private-window")

	var gotCtx context.Context
	var gotDirectory string
	var gotCommand []string
	var gotURL string
	runBrowserCommand = func(ctx context.Context, workingDirectory string, browserCommand []string, targetURL string) error {
		gotCtx = ctx
		gotDirectory = workingDirectory
		gotCommand = browserCommand
		gotURL = targetURL
		return nil
	}

	err := OpenURL(ctx, "test-dir", "https://example.com")
	require.NoError(t, err)
	assert.Same(t, ctx, gotCtx)
	assert.Equal(t, "test-dir", gotDirectory)
	assert.Equal(t, []string{"custom-browser", "--private-window"}, gotCommand)
	assert.Equal(t, "https://example.com", gotURL)
}

func TestOpenURLUsesShellForQuotedBrowserCommand(t *testing.T) {
	original := runBrowserCommand
	t.Cleanup(func() {
		runBrowserCommand = original
	})

	ctx := env.Set(t.Context(), browserEnvVar, `open -a "Google Chrome"`)

	var gotCommand []string
	runBrowserCommand = func(ctx context.Context, workingDirectory string, browserCommand []string, targetURL string) error {
		gotCommand = browserCommand
		return nil
	}

	err := OpenURL(ctx, ".", "https://example.com")
	require.NoError(t, err)

	if runtime.GOOS == "windows" {
		assert.Equal(t, []string{"cmd", "/c", `open -a "Google Chrome"`}, gotCommand)
	} else {
		assert.Equal(t, []string{"sh", "-c", `open -a "Google Chrome"`}, gotCommand)
	}
}

func TestParseBrowserCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"simple", "firefox", []string{"firefox"}},
		{"with flags", "firefox --private-window", []string{"firefox", "--private-window"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseBrowserCommand(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestParseBrowserCommandWithQuotes(t *testing.T) {
	got := parseBrowserCommand(`open -a "Google Chrome"`)
	shell := "sh"
	flag := "-c"
	if runtime.GOOS == "windows" {
		shell = "cmd"
		flag = "/c"
	}
	assert.Equal(t, []string{shell, flag, `open -a "Google Chrome"`}, got)
}

func TestIsDisabled(t *testing.T) {
	assert.False(t, IsDisabled(t.Context()))

	ctx := env.Set(t.Context(), browserEnvVar, disabledBrowser)
	assert.True(t, IsDisabled(ctx))
}
