package storage

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// holdLockEnvVar makes the test binary re-execute itself as a lock holder, so
// the contention is between two processes. An flock is shared by every
// descriptor in the process that took it, so a second in-process acquisition
// would succeed and prove nothing.
const holdLockEnvVar = "DATABRICKS_TEST_HOLD_TOKEN_STORE_LOCK"

func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

func TestLockTokenStoreCreatesLockFile(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)

	unlock, err := LockTokenStore(t.Context())
	require.NoError(t, err)
	defer unlock()

	assert.FileExists(t, filepath.Join(home, tokenStoreLockFilePath))
}

func TestLockTokenStoreIsReleasedByUnlock(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)

	unlock, err := LockTokenStore(t.Context())
	require.NoError(t, err)
	unlock()

	unlock, err = LockTokenStore(t.Context())
	require.NoError(t, err)
	unlock()
}

func TestLockTokenStoreWaitsForAnotherProcess(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)

	cmd := exec.Command(os.Args[0], "-test.run=^TestHoldsTokenStoreLockHelper$")
	cmd.Env = append(os.Environ(), holdLockEnvVar+"=1", "HOME="+home, "USERPROFILE="+home)
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())
	defer func() {
		stdin.Close()
		_ = cmd.Wait()
	}()

	// The helper prints this line once it holds the lock.
	scanner := bufio.NewScanner(stdout)
	require.True(t, scanner.Scan(), "helper process did not report holding the lock")
	require.Equal(t, "locked", scanner.Text())

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err = LockTokenStore(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// TestHoldsTokenStoreLockHelper is the child half of
// TestLockTokenStoreWaitsForAnotherProcess. It holds the lock until its stdin
// is closed, and is skipped during a normal test run.
func TestHoldsTokenStoreLockHelper(t *testing.T) {
	if os.Getenv(holdLockEnvVar) != "1" {
		t.Skip("helper process for TestLockTokenStoreWaitsForAnotherProcess")
	}

	unlock, err := LockTokenStore(t.Context())
	require.NoError(t, err)
	defer unlock()

	_, err = os.Stdout.WriteString("locked\n")
	require.NoError(t, err)

	// Block until the parent closes stdin.
	_, _ = os.Stdin.Read(make([]byte, 1))
}
