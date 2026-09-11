//go:build unix

package process

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithProcessGroupReapsGrandchild verifies that cancelling the context kills
// the whole process group, not just the direct child. The shell (the group
// leader) backgrounds a long sleep — a grandchild of the test process — and
// records its PID; after cancellation that grandchild must be gone.
func TestWithProcessGroupReapsGrandchild(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	// $1 is pidFile: background a sleep, record its PID, then wait on it so the
	// shell stays alive as the group leader until the group is signalled.
	script := []string{"sh", "-c", `sleep 300 & echo $! > "$1"; wait`, "sh", pidFile}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = Background(ctx, script, WithProcessGroup())
	}()

	grandchildPid := waitForPid(t, pidFile)

	cancel()

	select {
	case <-done:
	case <-time.After(processGroupGracePeriod + 5*time.Second):
		t.Fatal("Background did not return after context cancellation")
	}

	// The grandchild inherited the leader's group, so the group SIGTERM reaches
	// it directly. Poll briefly to let the kernel deliver the signal and reap it.
	assert.Eventually(t, func() bool {
		return errors.Is(syscall.Kill(grandchildPid, 0), syscall.ESRCH)
	}, 5*time.Second, 20*time.Millisecond, "grandchild %d was orphaned, not reaped", grandchildPid)
}

// TestWithProcessGroupReapsGrandchildAfterEscalation covers the SIGKILL
// escalation path: a leader that ignores SIGTERM (trap ” TERM). The group
// SIGTERM from Cancel does nothing, so WaitDelay expires and Go SIGKILLs the
// leader PID only — leaving the grandchild that this option exists to reap. The
// post-Wait group sweep (reapProcessGroup) must SIGKILL the whole group so the
// grandchild does not survive.
func TestWithProcessGroupReapsGrandchildAfterEscalation(t *testing.T) {
	// Shorten the grace period so the WaitDelay escalation fires quickly.
	orig := processGroupGracePeriod
	processGroupGracePeriod = 500 * time.Millisecond
	t.Cleanup(func() { processGroupGracePeriod = orig })

	ctx, cancel := context.WithCancel(t.Context())

	pidFile := filepath.Join(t.TempDir(), "grandchild.pid")
	// The leader ignores SIGTERM, so only the escalation can stop it; the sleep
	// grandchild inherits the group but not the trap.
	script := []string{"sh", "-c", `trap '' TERM; sleep 300 & echo $! > "$1"; wait`, "sh", pidFile}

	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = Background(ctx, script, WithProcessGroup())
	}()

	grandchildPid := waitForPid(t, pidFile)

	cancel()

	select {
	case <-done:
	case <-time.After(processGroupGracePeriod + 10*time.Second):
		t.Fatal("Background did not return after escalation")
	}

	assert.Eventually(t, func() bool {
		return errors.Is(syscall.Kill(grandchildPid, 0), syscall.ESRCH)
	}, 5*time.Second, 20*time.Millisecond,
		"grandchild %d survived the SIGKILL escalation (re-orphaned)", grandchildPid)
}

// waitForPid waits for the shell to write the grandchild PID and returns it.
func waitForPid(t *testing.T, pidFile string) int {
	t.Helper()
	var pid int
	require.Eventually(t, func() bool {
		b, err := os.ReadFile(pidFile)
		if err != nil {
			return false
		}
		pid, err = strconv.Atoi(strings.TrimSpace(string(b)))
		return err == nil && pid > 0
	}, 5*time.Second, 20*time.Millisecond, "grandchild PID was never recorded")
	return pid
}
