//go:build unix

package process

import (
	"context"
	"errors"
	"os"
	"os/exec"
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

func TestBackgroundInGroupForwardsInterrupt(t *testing.T) {
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	helper := exec.Command(os.Args[0], "-test.run=^TestBackgroundInGroupInterruptHelper$")
	helper.Env = append(os.Environ(), interruptHelperEnv+"="+pidFile)
	require.NoError(t, helper.Start())

	grandchildPid := waitForPid(t, pidFile)
	// A Ctrl-C in the terminal signals the helper, which is in the foreground
	// group, but not the child, which has its own group.
	require.NoError(t, helper.Process.Signal(os.Interrupt))

	waitErr := make(chan error, 1)
	go func() { waitErr <- helper.Wait() }()
	select {
	case err := <-waitErr:
		exitErr, ok := errors.AsType[*exec.ExitError](err)
		require.True(t, ok, "helper exited with %v", err)
		status := exitErr.Sys().(syscall.WaitStatus)
		assert.True(t, status.Signaled(), "the helper must end as the interrupt would have, got %v", err)
		assert.Equal(t, syscall.SIGINT, status.Signal())
	case <-time.After(processGroupGracePeriod + 5*time.Second):
		_ = helper.Process.Kill()
		t.Fatal("the interrupt did not end the helper")
	}

	assert.Eventually(t, func() bool {
		return errors.Is(syscall.Kill(grandchildPid, 0), syscall.ESRCH)
	}, 5*time.Second, 20*time.Millisecond, "grandchild %d outlived the interrupt", grandchildPid)
}

// TestBackgroundInGroupReapsGrandchildAfterEscalation covers the SIGKILL
// escalation for a child that leads its own session: the final group SIGKILL
// must reach the grandchild of a leader that ignores SIGTERM.
func TestBackgroundInGroupReapsGrandchildAfterEscalation(t *testing.T) {
	orig := processGroupGracePeriod
	processGroupGracePeriod = 500 * time.Millisecond
	t.Cleanup(func() { processGroupGracePeriod = orig })

	ctx, cancel := context.WithCancel(t.Context())
	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	script := []string{"sh", "-c", `trap '' TERM; sleep 300 & echo $! > "$1"; wait`, "sh", pidFile}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = BackgroundInGroup(ctx, script)
	}()

	grandchildPid := waitForPid(t, pidFile)
	cancel()

	select {
	case <-done:
	case <-time.After(processGroupGracePeriod + 10*time.Second):
		t.Fatal("BackgroundInGroup did not return after escalation")
	}

	assert.Eventually(t, func() bool {
		return errors.Is(syscall.Kill(grandchildPid, 0), syscall.ESRCH)
	}, 5*time.Second, 20*time.Millisecond,
		"grandchild %d survived the SIGKILL escalation (re-orphaned)", grandchildPid)
}
