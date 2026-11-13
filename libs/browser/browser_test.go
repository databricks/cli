package browser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDisabled(t *testing.T) {
	tests := []struct {
		browser  string
		expected bool
	}{
		{"", false},
		{"none", true},
		{"firefox", false},
	}

	for _, tt := range tests {
		t.Run(tt.browser, func(t *testing.T) {
			ctx := env.Set(t.Context(), "BROWSER", tt.browser)
			assert.Equal(t, tt.expected, IsDisabled(ctx))
		})
	}
}

func TestOpenWithBrowserNonePrintsURL(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = env.Set(ctx, "BROWSER", "none")

	require.NoError(t, Open(ctx, "https://example.com/resource"))

	assert.Contains(t, stderr.String(), "Open this URL in your browser:")
	assert.Contains(t, stderr.String(), "https://example.com/resource")
}

// openerChildEnv marks the child process that the tests below start as the browser command.
const openerChildEnv = "LIBS_BROWSER_TEST_OPENER"

// TestMain lets the tests use this test binary as a browser command that writes to stdout, as a launcher can.
func TestMain(m *testing.M) {
	if os.Getenv(openerChildEnv) != "" {
		os.Args = []string{os.Args[0], "-test.run=^TestOpenerChild$"}
	}
	m.Run()
}

// TestOpenerChild is the browser command. It runs only in the child process that TestMain starts.
func TestOpenerChild(t *testing.T) {
	if os.Getenv(openerChildEnv) == "" {
		t.Skip("runs only as the browser command of the tests below")
	}
	fmt.Print("opener stdout")
}

// runOpener calls open with this test binary as the browser command, and returns what the command wrote to stdout and
// to stderr. The process streams are replaced for the call.
func runOpener(t *testing.T, open func(context.Context, string) error) (stdout, stderr string, err error) {
	t.Helper()
	t.Setenv(openerChildEnv, "1")
	dir := t.TempDir()
	stdoutFile, err := os.Create(filepath.Join(dir, "stdout"))
	require.NoError(t, err)
	defer stdoutFile.Close()
	stderrFile, err := os.Create(filepath.Join(dir, "stderr"))
	require.NoError(t, err)
	defer stderrFile.Close()

	prevStdout, prevStderr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = stdoutFile, stderrFile
	ctx := env.Set(t.Context(), "BROWSER", os.Args[0])
	err = open(ctx, "https://example.com/login")
	os.Stdout, os.Stderr = prevStdout, prevStderr

	gotStdout, readErr := os.ReadFile(stdoutFile.Name())
	require.NoError(t, readErr)
	gotStderr, readErr := os.ReadFile(stderrFile.Name())
	require.NoError(t, readErr)
	return string(gotStdout), string(gotStderr), err
}

// TestOpenOutputToStderrSendsCommandStdoutToStderr checks that the output of a custom browser command goes to stderr, so
// that it does not land before a JSON document on stdout.
func TestOpenOutputToStderrSendsCommandStdoutToStderr(t *testing.T) {
	stdout, stderr, err := runOpener(t, OpenOutputToStderr)
	require.NoError(t, err)
	assert.Empty(t, stdout)
	assert.Contains(t, stderr, "opener stdout")
}

// TestOpenKeepsCommandStdoutOnStdout checks that Open, which other commands use, still sends the command's stdout to
// stdout.
func TestOpenKeepsCommandStdoutOnStdout(t *testing.T) {
	stdout, stderr, err := runOpener(t, Open)
	require.NoError(t, err)
	assert.Contains(t, stdout, "opener stdout")
	assert.NotContains(t, stderr, "opener stdout")
}
