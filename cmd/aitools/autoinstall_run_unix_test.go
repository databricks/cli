//go:build unix

package aitools

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/process"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// autoInstallSignalHelperEnv makes TestMain run the hook in a helper process,
// which the test signals during the gate lookup. The CLI ends by the signal,
// so it cannot run in the test process.
const autoInstallSignalHelperEnv = "AITOOLS_TEST_SIGNAL_HELPER"

// loginSignalHelperEnv makes TestMain run a login that asks the question in a
// helper process, which the test signals at the question.
const loginSignalHelperEnv = "AITOOLS_TEST_LOGIN_SIGNAL_HELPER"

// sigintIgnoredAtStart is whether the test process started with SIGINT
// ignored. It is read before any test catches SIGINT, which clears the record.
var sigintIgnoredAtStart = signal.Ignored(syscall.SIGINT)

// gateLookupStarted is what the helper prints when its gate lookup starts.
const gateLookupStarted = "looking the gate up"

func init() {
	runTestHelper = runAutoInstallSignalHelperProcess
}

// runAutoInstallSignalHelperProcess runs the hook when
// TestAutoInstallEndsByTheSignal re-executes the test binary as the helper,
// and the login when TestAskAtLoginEndsByTheSignal does.
func runAutoInstallSignalHelperProcess() {
	if os.Getenv(loginSignalHelperEnv) != "" {
		runLoginSignalHelper()
		os.Exit(0)
	}
	if os.Getenv(autoInstallSignalHelperEnv) == "" {
		return
	}
	// The root command ends the CLI by the signal that the hook returns;
	// exiting normally fails the test.
	if sig := runAutoInstallSignalHelper(); sig != nil {
		process.Raise(sig)
	}
	os.Exit(0)
}

func runAutoInstallSignalHelper() os.Signal {
	// The helper is a new process: it has the real clock, not the test's.
	now = func() time.Time { return autoInstallTestNow }
	// Only the signal ends the lookup.
	gateTimeout = time.Hour
	// A lookup that waits for the manifest until it is stopped.
	gateOn = func(ctx context.Context, _ string) (bool, error) {
		fmt.Fprintln(os.Stderr, gateLookupStarted)
		<-ctx.Done()
		return false, ctx.Err()
	}
	// TestMain has no *testing.T, so t.Context() is unavailable here.
	//nolint:gocritic
	ctx := context.Background()
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText, cmdio.FakeTTYReader(os.Stdin), cmdio.FakeTTY(io.Discard), cmdio.FakeTTY(os.Stderr), "", ""))
	ctx = telemetry.WithNewLogger(ctx)
	return runAutoInstallHook(ctx)
}

// runLoginSignalHelper runs a login that asks the question through the root
// command, as main.go wires it, with stdin a pipe that nobody writes to. The
// root command ends the process by the signal; returning fails the test.
func runLoginSignalHelper() {
	// The helper is a new process: it has the real clock, not the test's.
	now = func() time.Time { return autoInstallTestNow }
	autoinstall.Run, autoinstall.AskAtLogin = AutoInstall, AskAtLogin
	// TestMain has no *testing.T, so t.Context() is unavailable here.
	//nolint:gocritic
	ctx := context.Background()
	cli := root.NewTree(ctx)
	cli.Add(newLoginStandIn())
	cli.SetIn(cmdio.FakeTTYReader(os.Stdin))
	cli.SetOut(cmdio.FakeTTY(io.Discard))
	cli.SetErr(cmdio.FakeTTY(os.Stderr))
	_ = cli.Execute(ctx, []string{"auth", "login"})
}

// Ctrl+C at the question of the login ends the CLI by the signal, only after
// the question was recorded as asked: the consent stays unknown and no later
// login asks again. Nothing is installed.
func TestAskAtLoginEndsByTheSignal(t *testing.T) {
	// The helper inherits a SIGINT ignored at start, as in a background job
	// of a shell without job control, and the CLI then rightly does not end
	// by it.
	if sigintIgnoredAtStart {
		t.Skip("SIGINT was ignored when the test started")
	}
	home := setupAutoInstallHome(t, unaskedState)
	helper := exec.Command(os.Args[0])
	helper.Env = append(os.Environ(), loginSignalHelperEnv+"=1")
	stdin, err := helper.StdinPipe()
	require.NoError(t, err)
	t.Cleanup(func() { _ = stdin.Close() })
	stderr, err := helper.StderrPipe()
	require.NoError(t, err)
	require.NoError(t, helper.Start())

	output := waitForOutput(t, stderr, cursorQuestion)
	require.NoError(t, helper.Process.Signal(syscall.SIGINT))

	waitErr := make(chan error, 1)
	go func() { waitErr <- helper.Wait() }()
	select {
	case err := <-waitErr:
		exitErr, ok := errors.AsType[*exec.ExitError](err)
		require.True(t, ok, "the helper exited with %v; it printed:\n%s", err, output)
		status := exitErr.Sys().(syscall.WaitStatus)
		assert.True(t, status.Signaled(), "the helper must end by the signal, got %v", err)
		assert.Equal(t, syscall.SIGINT, status.Signal())
	case <-time.After(10 * time.Second):
		_ = helper.Process.Kill()
		t.Fatal("the signal did not end the helper")
	}

	state, err := autoinstall.Load(t.Context())
	require.NoError(t, err)
	assert.Equal(t, autoinstall.State{ConsentAsked: true, ID: "treatment-machine"}, state)
	assert.NoDirExists(t, filepath.Join(home, ".cursor", "skills"))
}

// A signal during the gate lookup ends the CLI by that signal, so that the
// shell stops the script or loop that ran the command, and only after the
// version is recorded: the next command does not look the gate up again.
func TestAutoInstallEndsByTheSignal(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
			// The helper inherits a SIGHUP ignored at start, as under nohup,
			// and then rightly leaves it alone.
			if !slices.Contains(process.InterruptSignals(), os.Signal(sig)) {
				t.Skipf("%v was ignored when the test started", sig)
			}
			setupAutoInstallHome(t, "")
			t.Setenv("TERM", "xterm")
			helper := exec.Command(os.Args[0])
			helper.Env = append(os.Environ(), autoInstallSignalHelperEnv+"=1")
			stderr, err := helper.StderrPipe()
			require.NoError(t, err)
			require.NoError(t, helper.Start())

			output := waitForOutput(t, stderr, gateLookupStarted)
			require.NoError(t, helper.Process.Signal(sig))

			waitErr := make(chan error, 1)
			go func() { waitErr <- helper.Wait() }()
			select {
			case err := <-waitErr:
				exitErr, ok := errors.AsType[*exec.ExitError](err)
				require.True(t, ok, "the helper exited with %v; it printed:\n%s", err, output)
				status := exitErr.Sys().(syscall.WaitStatus)
				assert.True(t, status.Signaled(), "the helper must end by the signal, got %v", err)
				assert.Equal(t, sig, status.Signal())
			case <-time.After(10 * time.Second):
				_ = helper.Process.Kill()
				t.Fatal("the signal did not end the helper")
			}

			state, err := autoinstall.Load(t.Context())
			require.NoError(t, err)
			assert.Equal(t, autoinstall.State{LastVersion: "1.30.0", Attempts: autoinstall.MaxAttempts, Consented: true, ID: "treatment-machine"}, state)
		})
	}
}

// waitForOutput reads r until it has printed want, and returns what it read.
func waitForOutput(t *testing.T, r io.Reader, want string) string {
	t.Helper()
	found := make(chan struct{})
	var out bytes.Buffer
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := r.Read(buf)
			out.Write(buf[:n])
			if bytes.Contains(out.Bytes(), []byte(want)) {
				close(found)
				return
			}
			if err != nil {
				return
			}
		}
	}()
	select {
	case <-found:
		return out.String()
	case <-time.After(10 * time.Second):
		t.Fatalf("the helper did not print %q", want)
		return ""
	}
}
