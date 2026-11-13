package browser

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	browserpkg "github.com/pkg/browser"

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

const (
	// openerChildEnv marks the child process that the tests below start as the browser command.
	openerChildEnv = "LIBS_BROWSER_TEST_OPENER"
	// openerBackgroundEnv, when set in the browser command, makes it start a sleeper and return, as a launcher does when
	// it backgrounds the browser. Its value is the file the sleeper writes when it ends.
	openerBackgroundEnv = "LIBS_BROWSER_TEST_BACKGROUND"
	// openerSleeperEnv marks the child that openerBackgroundEnv starts, and names the file it writes.
	openerSleeperEnv = "LIBS_BROWSER_TEST_SLEEPER"
	// openerSleepTime is how long the sleeper runs. It is short, so that the tests end soon even when the wait is wrong.
	openerSleepTime = 3 * time.Second
)

// TestMain lets the tests use this test binary as a browser command that writes to stdout, as a launcher can.
func TestMain(m *testing.M) {
	switch {
	case os.Getenv(openerSleeperEnv) != "":
		os.Args = []string{os.Args[0], "-test.run=^TestOpenerSleeper$"}
	case os.Getenv(openerChildEnv) != "":
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
	sleepFile := os.Getenv(openerBackgroundEnv)
	if sleepFile == "" {
		return
	}
	// The sleeper inherits this process's output, as a browser started in the background does, and outlives the command.
	sleeper := exec.Command(os.Args[0], "-test.run=^TestOpenerSleeper$")
	sleeper.Env = append(os.Environ(), openerSleeperEnv+"="+sleepFile)
	sleeper.Stdout = os.Stdout
	sleeper.Stderr = os.Stderr
	if err := sleeper.Start(); err != nil {
		t.Fatal(err)
	}
}

// TestOpenerSleeper is the child that openerBackgroundEnv starts. It runs only in that child process.
func TestOpenerSleeper(t *testing.T) {
	file := os.Getenv(openerSleeperEnv)
	if file == "" {
		t.Skip("runs only as the child of the browser command")
	}
	time.Sleep(openerSleepTime)
	_ = os.WriteFile(file, nil, 0o600)
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

// TestOpenOutputToStderrReturnsWhenCommandBackgroundsChild checks that a browser command that starts a child and returns
// does not hold up the caller. The child outlives the command, as a launcher that backgrounds the browser does, and writes
// its file only when it ends. The file must not exist when the call returns.
func TestOpenOutputToStderrReturnsWhenCommandBackgroundsChild(t *testing.T) {
	t.Setenv(openerChildEnv, "1")
	sleepFile := filepath.Join(t.TempDir(), "sleeper-done")
	t.Setenv(openerBackgroundEnv, sleepFile)
	ctx := env.Set(t.Context(), "BROWSER", os.Args[0])

	require.NoError(t, OpenOutputToStderr(ctx, "https://example.com/login"))
	_, err := os.Stat(sleepFile)
	assert.ErrorIs(t, err, fs.ErrNotExist, "the call waited for the child that the browser command started")
}

// TestOpenDefaultOverlappingOpensRestoreStreams checks that two default opens that run at once, as sign-in can start
// one while add --login starts another, leave the streams of pkg/browser as they were. The PATH is empty, so the
// opener cannot start a browser: the opens fail at once, after they have swapped the streams. The race detector reports
// the unsynchronized swaps when two opens overlap.
func TestOpenDefaultOverlappingOpensRestoreStreams(t *testing.T) {
	t.Setenv("BROWSER", "")
	t.Setenv("PATH", t.TempDir())
	stdout, stderr := browserpkg.Stdout, browserpkg.Stderr

	var opens sync.WaitGroup
	for range 2 {
		opens.Go(func() {
			_ = OpenOutputToStderr(t.Context(), "https://example.com/login")
		})
	}
	opens.Wait()

	assert.Equal(t, stdout, browserpkg.Stdout)
	assert.Equal(t, stderr, browserpkg.Stderr)
}
