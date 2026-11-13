package browser

import (
	"context"
	"fmt"
	"io"
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
	// openerStdinEnv, when set in the browser command, makes it read up to 16 bytes of its stdin and print them.
	openerStdinEnv = "LIBS_BROWSER_TEST_STDIN"
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
	if os.Getenv(openerStdinEnv) != "" {
		buf := make([]byte, 16)
		n, _ := os.Stdin.Read(buf)
		fmt.Fprintf(os.Stderr, "stdin: %q", buf[:n])
	}
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
	defer func() { os.Stdout, os.Stderr = prevStdout, prevStderr }()
	ctx := env.Set(t.Context(), "BROWSER", os.Args[0])
	err = open(ctx, "https://example.com/login")

	gotStdout, readErr := os.ReadFile(stdoutFile.Name())
	require.NoError(t, readErr)
	gotStderr, readErr := os.ReadFile(stderrFile.Name())
	require.NoError(t, readErr)
	return string(gotStdout), string(gotStderr), err
}

// replaceDefaultLauncher sets the launcher of the default opener to fn for the test.
func replaceDefaultLauncher(t *testing.T, fn func(url string) error) {
	t.Helper()
	prev := openDefaultURL
	t.Cleanup(func() { openDefaultURL = prev })
	openDefaultURL = fn
}

// waitForSleeper registers a cleanup that waits until the sleeper that a test's browser command started has ended, so
// that no child outlives the test.
func waitForSleeper(t *testing.T, sleepFile string) {
	t.Helper()
	t.Cleanup(func() {
		assert.Eventually(t, func() bool {
			_, err := os.Stat(sleepFile)
			return err == nil
		}, 5*openerSleepTime, 10*time.Millisecond, "the sleeper did not end")
	})
}

// TestOpenCommandOutput checks where the output of a custom browser command goes. OpenOutputToStderr sends it to stderr,
// so that it does not land before a JSON document on stdout. Open, which other commands use, keeps it on stdout.
func TestOpenCommandOutput(t *testing.T) {
	tests := []struct {
		name       string
		open       func(context.Context, string) error
		wantStdout bool
	}{
		{name: "OpenOutputToStderr sends the output to stderr", open: OpenOutputToStderr, wantStdout: false},
		{name: "Open keeps the output on stdout", open: Open, wantStdout: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runOpener(t, tt.open)
			require.NoError(t, err)
			if tt.wantStdout {
				assert.Contains(t, stdout, "opener stdout")
				assert.NotContains(t, stderr, "opener stdout")
				return
			}
			assert.Empty(t, stdout)
			assert.Contains(t, stderr, "opener stdout")
		})
	}
}

// TestOpenDoesNotWaitForLauncherChild checks that the default opener returns when its launcher returns, although the
// browser the launcher started is still running. The launcher is a stub that runs this test binary the way pkg/browser's
// runCmd runs xdg-open: with the package's Stdout and Stderr, and waiting only for the command. The sleeper is the
// browser, and it writes sleepFile when it ends. Neither open may wait for it.
func TestOpenDoesNotWaitForLauncherChild(t *testing.T) {
	t.Setenv(openerChildEnv, "1")
	sleepFile := filepath.Join(t.TempDir(), "sleeper-done")
	t.Setenv(openerBackgroundEnv, sleepFile)
	waitForSleeper(t, sleepFile)
	replaceDefaultLauncher(t, func(url string) error {
		cmd := exec.Command(os.Args[0], url)
		cmd.Stdout = browserpkg.Stdout
		cmd.Stderr = browserpkg.Stderr
		return cmd.Run()
	})
	ctx := env.Set(t.Context(), "BROWSER", "")

	for range 2 {
		require.NoError(t, Open(ctx, "https://example.com/login"))
		_, err := os.Stat(sleepFile)
		assert.ErrorIs(t, err, fs.ErrNotExist, "the open waited for the browser that the launcher started")
	}
}

// TestOpenOutputToStderrReturnsWhenCommandBackgroundsChild checks that a browser command that starts a child and returns
// does not hold up the caller. The child outlives the command, as a launcher that backgrounds the browser does, and writes
// its file only when it ends. The file must not exist when the call returns.
func TestOpenOutputToStderrReturnsWhenCommandBackgroundsChild(t *testing.T) {
	t.Setenv(openerChildEnv, "1")
	sleepFile := filepath.Join(t.TempDir(), "sleeper-done")
	t.Setenv(openerBackgroundEnv, sleepFile)
	waitForSleeper(t, sleepFile)
	ctx := env.Set(t.Context(), "BROWSER", os.Args[0])

	require.NoError(t, OpenOutputToStderr(ctx, "https://example.com/login"))
	_, err := os.Stat(sleepFile)
	assert.ErrorIs(t, err, fs.ErrNotExist, "the call waited for the child that the browser command started")
}

// TestOpenDefaultOverlappingOpensRestoreStreams checks that two default opens that run at once, as sign-in can start
// one while add --login starts another, leave the streams of pkg/browser as they were. The launcher is a stub that
// starts nothing, so no browser opens on any OS. The race detector reports the unsynchronized swaps when two opens
// overlap.
func TestOpenDefaultOverlappingOpensRestoreStreams(t *testing.T) {
	t.Setenv("BROWSER", "")
	replaceDefaultLauncher(t, func(string) error { return nil })
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

// TestOpenWithoutStdinGivesCommandNoInput checks that a browser command opened without stdin reads none, so it cannot read
// the stream on this process's stdin, while the other openers give it that stream. The command writes what it read to
// stderr.
func TestOpenWithoutStdinGivesCommandNoInput(t *testing.T) {
	t.Setenv(openerStdinEnv, "1")
	// stdinWith makes the stdin of this process a pipe that holds data and stays open, so a command that inherits it
	// would wait for more input than the 16 bytes it reads.
	stdinWith := func(data string) {
		r, w, err := os.Pipe()
		require.NoError(t, err)
		t.Cleanup(func() {
			r.Close()
			w.Close()
		})
		_, err = io.WriteString(w, data)
		require.NoError(t, err)
		prevStdin := os.Stdin
		os.Stdin = r
		t.Cleanup(func() { os.Stdin = prevStdin })
	}

	stdinWith("protocol")
	_, stderr, err := runOpener(t, OpenOutputToStderrWithoutStdin)
	require.NoError(t, err)
	assert.Contains(t, stderr, `stdin: ""`)

	stdinWith("protocol")
	_, stderr, err = runOpener(t, OpenOutputToStderr)
	require.NoError(t, err)
	assert.Contains(t, stderr, `stdin: "protocol"`)
}
