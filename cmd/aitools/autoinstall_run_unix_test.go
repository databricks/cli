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
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/process"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// autoInstallSignalHelperEnv makes TestMain run the hook in a helper process,
// which the test signals at the step the value names: "question" or "lookup".
// The CLI ends by the signal, so it cannot run in the test process.
const autoInstallSignalHelperEnv = "AITOOLS_TEST_SIGNAL_HELPER"

// gateLookupStarted is what the helper prints when its gate lookup starts.
const gateLookupStarted = "looking the gate up"

func init() {
	runTestHelper = runAutoInstallSignalHelperProcess
}

// runAutoInstallSignalHelperProcess runs the hook when
// TestAutoInstallEndsByTheSignal re-executes the test binary as the helper.
func runAutoInstallSignalHelperProcess() {
	step := os.Getenv(autoInstallSignalHelperEnv)
	if step == "" {
		return
	}
	// The root command ends the CLI by the signal that the hook returns;
	// exiting normally fails the test.
	if sig := runAutoInstallSignalHelper(step); sig != nil {
		process.Raise(sig)
	}
	os.Exit(0)
}

func runAutoInstallSignalHelper(step string) os.Signal {
	// The helper is a new process: it has the real clock, not the test's.
	now = func() time.Time { return autoInstallTestNow }
	if step == "lookup" {
		// Only the signal ends the lookup.
		gateTimeout = time.Hour
		// A lookup that waits for the manifest until it is stopped.
		gateOn = func(ctx context.Context, _ string) (bool, error) {
			fmt.Fprintln(os.Stderr, gateLookupStarted)
			<-ctx.Done()
			return false, ctx.Err()
		}
	}
	// TestMain has no *testing.T, so t.Context() is unavailable here.
	//nolint:gocritic
	ctx := context.Background()
	// stdin is a pipe that nobody writes to, so the question waits for an
	// answer.
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText, cmdio.FakeTTYReader(os.Stdin), cmdio.FakeTTY(io.Discard), cmdio.FakeTTY(os.Stderr), "", ""))
	ctx = telemetry.WithNewLogger(ctx)
	return runAutoInstallHook(ctx)
}

// A signal at the question or during the gate lookup ends the CLI by that
// signal, so that the shell stops the script or loop that ran the command,
// and only after the version is recorded: the next command neither asks nor
// looks the gate up again.
func TestAutoInstallEndsByTheSignal(t *testing.T) {
	tests := []struct {
		step      string
		sig       syscall.Signal
		wantState autoinstall.State
	}{
		{"question", syscall.SIGINT, autoinstall.State{LastVersion: "1.30.0", ID: "treatment-machine"}},
		{"lookup", syscall.SIGINT, autoinstall.State{LastVersion: "1.30.0", Attempts: autoinstall.MaxAttempts, ID: "treatment-machine"}},
		{"lookup", syscall.SIGTERM, autoinstall.State{LastVersion: "1.30.0", Attempts: autoinstall.MaxAttempts, ID: "treatment-machine"}},
		{"lookup", syscall.SIGHUP, autoinstall.State{LastVersion: "1.30.0", Attempts: autoinstall.MaxAttempts, ID: "treatment-machine"}},
	}
	for _, tc := range tests {
		t.Run(tc.step+"/"+tc.sig.String(), func(t *testing.T) {
			// The helper inherits a SIGHUP ignored at start, as under nohup,
			// and then rightly leaves it alone.
			if !slices.Contains(process.InterruptSignals(), os.Signal(tc.sig)) {
				t.Skipf("%v was ignored when the test started", tc.sig)
			}
			setupAutoInstallHome(t, "")
			t.Setenv("TERM", "xterm")
			helper := exec.Command(os.Args[0])
			helper.Env = append(os.Environ(), autoInstallSignalHelperEnv+"="+tc.step)
			stdin, err := helper.StdinPipe()
			require.NoError(t, err)
			t.Cleanup(func() { _ = stdin.Close() })
			stderr, err := helper.StderrPipe()
			require.NoError(t, err)
			require.NoError(t, helper.Start())

			want := gateLookupStarted
			if tc.step == "question" {
				want = "Install Databricks AI tools for Cursor? [Y/n]: "
			}
			output := waitForOutput(t, stderr, want)
			require.NoError(t, helper.Process.Signal(tc.sig))

			waitErr := make(chan error, 1)
			go func() { waitErr <- helper.Wait() }()
			select {
			case err := <-waitErr:
				exitErr, ok := errors.AsType[*exec.ExitError](err)
				require.True(t, ok, "the helper exited with %v; it printed:\n%s", err, output)
				status := exitErr.Sys().(syscall.WaitStatus)
				assert.True(t, status.Signaled(), "the helper must end by the signal, got %v", err)
				assert.Equal(t, tc.sig, status.Signal())
			case <-time.After(10 * time.Second):
				_ = helper.Process.Kill()
				t.Fatal("the signal did not end the helper")
			}

			state, err := autoinstall.Load(t.Context())
			require.NoError(t, err)
			assert.Equal(t, tc.wantState, state)
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
