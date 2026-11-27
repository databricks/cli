//go:build unix

package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"slices"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// interruptHelperEnv makes TestBackgroundInGroupInterruptHelper run
// BackgroundInGroup in a helper process, which the test interrupts. Raising
// the signal again ends the process, so it cannot run in the test process.
const interruptHelperEnv = "PROCESS_TEST_INTERRUPT_HELPER_PIDFILE"

func TestBackgroundInGroupInterruptHelper(t *testing.T) {
	pidFile := os.Getenv(interruptHelperEnv)
	if pidFile == "" {
		t.Skip("run by TestBackgroundInGroupForwardsInterrupt")
	}
	_, _ = BackgroundInGroup(t.Context(), []string{"sh", "-c", `sleep 300 & echo $! > "$1"; wait`, "sh", pidFile})
	// The raised signal ends the process asynchronously.
	time.Sleep(5 * time.Second)
	t.Fatal("the interrupt did not end the process")
}

// raiseHelperEnv makes TestMain raise a signal in a helper process, which
// exits normally if Raise returns. The value names the signal: "SIGINT", which
// the helper also catches elsewhere, or "SIGHUP", which it started with
// ignored.
const raiseHelperEnv = "PROCESS_TEST_RAISE_HELPER"

// TestMain raises the signal when a Raise test re-executes the test binary as
// the helper.
func TestMain(m *testing.M) {
	switch os.Getenv(raiseHelperEnv) {
	case "":
		os.Exit(m.Run())
	case "SIGINT":
		// Another part of the process still catches it.
		signal.Notify(make(chan os.Signal, 1), os.Interrupt)
		Raise(os.Interrupt)
	case "SIGHUP":
		if slices.Contains(InterruptSignals(), os.Signal(syscall.SIGHUP)) {
			fmt.Fprintln(os.Stderr, "SIGHUP is caught although it was ignored at start")
			os.Exit(2)
		}
		Raise(syscall.SIGHUP)
	}
	os.Exit(0)
}

func TestRaiseEndsTheProcessBySignal(t *testing.T) {
	helper := exec.Command(os.Args[0])
	helper.Env = append(os.Environ(), raiseHelperEnv+"=SIGINT")
	err := helper.Run()
	exitErr, ok := errors.AsType[*exec.ExitError](err)
	require.True(t, ok, "helper exited with %v", err)
	status := exitErr.Sys().(syscall.WaitStatus)
	assert.True(t, status.Signaled(), "the helper must end by the signal, got %v", err)
	assert.Equal(t, syscall.SIGINT, status.Signal())
}

// A process started with SIGHUP ignored, as under nohup, does not catch it,
// and the signal does not end it: Raise returns.
func TestRaiseReturnsWhenSIGHUPWasIgnoredAtStart(t *testing.T) {
	// The shell starts the helper with SIGHUP ignored, as nohup does. Ignoring
	// it in the test process instead would outlast signal.Reset and reach the
	// helpers of later tests.
	helper := exec.Command("sh", "-c", `trap '' HUP; exec "$0"`, os.Args[0])
	helper.Env = append(os.Environ(), raiseHelperEnv+"=SIGHUP")
	out, err := helper.CombinedOutput()
	require.NoError(t, err, "helper printed:\n%s", out)
}

func TestBackgroundInGroupForwardsInterrupt(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
			// The helper inherits a SIGHUP ignored at start, as under nohup,
			// and then rightly leaves it alone.
			if !slices.Contains(InterruptSignals(), os.Signal(sig)) {
				t.Skipf("%v was ignored when the test started", sig)
			}
			testBackgroundInGroupForwards(t, sig)
		})
	}
}

func testBackgroundInGroupForwards(t *testing.T, sig syscall.Signal) {
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	helper := exec.Command(os.Args[0], "-test.run=^TestBackgroundInGroupInterruptHelper$")
	helper.Env = append(os.Environ(), interruptHelperEnv+"="+pidFile)
	require.NoError(t, helper.Start())

	grandchildPid := waitForPid(t, pidFile)
	// A Ctrl-C in the terminal, or the terminal closing, signals the helper,
	// which is in the foreground group, but not the child, which has its own
	// group.
	require.NoError(t, helper.Process.Signal(sig))

	waitErr := make(chan error, 1)
	go func() { waitErr <- helper.Wait() }()
	select {
	case err := <-waitErr:
		exitErr, ok := errors.AsType[*exec.ExitError](err)
		require.True(t, ok, "helper exited with %v", err)
		status := exitErr.Sys().(syscall.WaitStatus)
		assert.True(t, status.Signaled(), "the helper must end as the interrupt would have, got %v", err)
		assert.Equal(t, sig, status.Signal())
	case <-time.After(processGroupGracePeriod + 5*time.Second):
		_ = helper.Process.Kill()
		t.Fatal("the interrupt did not end the helper")
	}

	assert.Eventually(t, func() bool {
		return errors.Is(syscall.Kill(grandchildPid, 0), syscall.ESRCH)
	}, 5*time.Second, 20*time.Millisecond, "grandchild %d outlived the interrupt", grandchildPid)
}
