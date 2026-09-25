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
	"golang.org/x/oauth2"
)

// holdLockEnvVar makes the test binary re-execute itself as a lock holder, so
// the lock is also exercised across a real process boundary.
const holdLockEnvVar = "DATABRICKS_TEST_HOLD_TOKEN_STORE_LOCK"

// contendedWait is how long the tests wait on a lock they expect to be held.
const contendedWait = 100 * time.Millisecond

func setHome(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
}

// requireLockHeld fails the test unless the token store lock is held by
// someone else. Each call opens the lock file anew, and an open file
// description contends with every other one, including those in this process.
func requireLockHeld(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), contendedWait)
	defer cancel()
	unlock, err := lockTokenStore(ctx)
	if err == nil {
		unlock()
	}
	require.ErrorIs(t, err, context.DeadlineExceeded, "want the token store lock to be held")
}

// requireLockFree fails the test unless the token store lock can be taken.
func requireLockFree(t *testing.T) {
	t.Helper()
	unlock, err := lockTokenStoreWithTimeout(t.Context(), contendedWait)
	require.NoError(t, err, "want the token store lock to be free")
	unlock()
}

func TestLockTokenStoreCreatesLockFile(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)

	unlock, err := lockTokenStore(t.Context())
	require.NoError(t, err)
	defer unlock()

	assert.FileExists(t, filepath.Join(home, tokenStoreLockFilePath))
}

func TestLockTokenStoreIsReleasedByUnlock(t *testing.T) {
	setHome(t, t.TempDir())

	unlock, err := lockTokenStore(t.Context())
	require.NoError(t, err)
	requireLockHeld(t)
	unlock()

	requireLockFree(t)
}

func TestLockTokenStoreReturnsWhenContextIsDone(t *testing.T) {
	setHome(t, t.TempDir())
	unlock, err := lockTokenStore(t.Context())
	require.NoError(t, err)
	defer unlock()

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = lockTokenStore(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLockTokenStoreTimesOutBehindLiveHolder(t *testing.T) {
	setHome(t, t.TempDir())
	unlock, err := lockTokenStore(t.Context())
	require.NoError(t, err)
	defer unlock()

	_, err = lockTokenStoreWithTimeout(t.Context(), contendedWait)
	assert.ErrorContains(t, err, "timed out after 100ms waiting for another databricks process to release")
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

	requireLockHeld(t)
}

// TestHoldsTokenStoreLockHelper is the child half of
// TestLockTokenStoreWaitsForAnotherProcess. It holds the lock until its stdin
// is closed, and is skipped during a normal test run.
func TestHoldsTokenStoreLockHelper(t *testing.T) {
	if os.Getenv(holdLockEnvVar) != "1" {
		t.Skip("helper process for TestLockTokenStoreWaitsForAnotherProcess")
	}

	unlock, err := lockTokenStore(t.Context())
	require.NoError(t, err)
	defer unlock()

	_, err = os.Stdout.WriteString("locked\n")
	require.NoError(t, err)

	// Block until the parent closes stdin.
	_, _ = os.Stdin.Read(make([]byte, 1))
}

// TestSharedStoresTakeTokenStoreLock checks every store shared with other
// processes, as wrapped for U2M callers, takes the token store lock.
func TestSharedStoresTakeTokenStoreLock(t *testing.T) {
	setHome(t, t.TempDir())
	fileStore, err := NewFileStore(t.Context())
	require.NoError(t, err)
	arg := hostArg{key: "profile", hostKey: "https://host.test"}

	tests := []struct {
		name  string
		store Store
	}{
		{"file", fileStore},
		{"keyring", newTestStore(newFakeBackend())},
		{"plaintext for U2M", WrapForOAuthArgument(t.Context(), fileStore, StorageModePlaintext, arg)},
		{"secure for U2M", WrapForOAuthArgument(t.Context(), newTestStore(newFakeBackend()), StorageModeSecure, arg)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unlock, err := tt.store.Lock(t.Context())
			require.NoError(t, err)
			requireLockHeld(t)
			unlock()
			requireLockFree(t)
		})
	}
}

// gatedBackend is a keyring backend whose Set blocks until release is closed,
// modeling a keyring write that outlives the store's timeout.
type gatedBackend struct {
	*fakeBackend
	release chan struct{}
}

func (g gatedBackend) Set(service, account, secret string) error {
	<-g.release
	return g.fakeBackend.Set(service, account, secret)
}

func TestKeyringLockOutlivesTimedOutWrite(t *testing.T) {
	setHome(t, t.TempDir())
	backend := gatedBackend{fakeBackend: newFakeBackend(), release: make(chan struct{})}
	store := newTestStore(backend)

	unlock, err := store.Lock(t.Context())
	require.NoError(t, err)
	err = store.Put("profile", Entry{Token: &oauth2.Token{AccessToken: "late"}})
	var timeoutErr *TimeoutError
	require.ErrorAs(t, err, &timeoutErr)
	unlock()

	// The write is still running, so a new holder could be overwritten by it.
	requireLockHeld(t)

	close(backend.release)
	require.Eventually(t, func() bool {
		unlock, err := lockTokenStoreWithTimeout(t.Context(), contendedWait)
		if err != nil {
			return false
		}
		unlock()
		return true
	}, 5*time.Second, 10*time.Millisecond)
}
