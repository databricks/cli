//go:build unix

package process

import (
	"errors"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
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

// raiseHelperEnv makes TestMain raise SIGINT in a helper process, which exits
// normally if Raise returns.
const raiseHelperEnv = "PROCESS_TEST_RAISE_HELPER"

// TestMain raises the signal when TestRaiseEndsTheProcessBySignal re-executes
// the test binary as the helper.
func TestMain(m *testing.M) {
	if os.Getenv(raiseHelperEnv) == "" {
		os.Exit(m.Run())
	}
	Raise(os.Interrupt)
	os.Exit(0)
}

func TestRaiseEndsTheProcessBySignal(t *testing.T) {
	helper := exec.Command(os.Args[0])
	helper.Env = append(os.Environ(), raiseHelperEnv+"=1")
	err := helper.Run()
	exitErr, ok := errors.AsType[*exec.ExitError](err)
	require.True(t, ok, "helper exited with %v", err)
	status := exitErr.Sys().(syscall.WaitStatus)
	assert.True(t, status.Signaled(), "the helper must end by the signal, got %v", err)
	assert.Equal(t, syscall.SIGINT, status.Signal())
}

// A signal the process ignores does not end it, and Raise returns.
func TestRaiseReturnsWhenTheSignalIsIgnored(t *testing.T) {
	signal.Ignore(syscall.SIGUSR1)
	t.Cleanup(func() { signal.Reset(syscall.SIGUSR1) })
	Raise(syscall.SIGUSR1)
}

func TestBackgroundInGroupForwardsInterrupt(t *testing.T) {
	for _, sig := range []syscall.Signal{syscall.SIGINT, syscall.SIGHUP} {
		t.Run(sig.String(), func(t *testing.T) {
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
