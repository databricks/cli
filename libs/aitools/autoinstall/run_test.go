package autoinstall

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTryLockExcludesOtherRuns(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())

	unlock, ok := TryLock(ctx)
	require.True(t, ok)
	_, ok = TryLock(ctx)
	assert.False(t, ok, "a second run must not take a held lock")

	unlock()
	unlock, ok = TryLock(ctx)
	require.True(t, ok, "a released lock can be taken again")
	unlock()
}

func TestRecordKeepsConcurrentOptOut(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())

	// The 21 writers take the lock one at a time, and on a slow runner each
	// holds it long enough that the last ones would give up after the
	// default wait.
	orig := stateLockWait
	t.Cleanup(func() { stateLockWait = orig })
	stateLockWait = time.Minute

	// Without the state lock, a Record that read the state before the opt-out
	// was saved would write opted_out=false back.
	var wg sync.WaitGroup
	for range 20 {
		wg.Go(func() { assert.NoError(t, Record(ctx, "1.30.0", 1, "machine")) })
	}
	wg.Go(func() { assert.NoError(t, SetOptedOut(ctx, true)) })
	wg.Wait()

	s, err := Load(ctx)
	require.NoError(t, err)
	assert.True(t, s.OptedOut)
	assert.Equal(t, "1.30.0", s.LastVersion)
}

func TestSaveIDKeepsTheFirstID(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())

	id, err := SaveID(ctx, "first")
	require.NoError(t, err)
	assert.Equal(t, "first", id)
	id, err = SaveID(ctx, "second")
	require.NoError(t, err)
	assert.Equal(t, "first", id, "the stored ID is returned")

	s, err := Load(ctx)
	require.NoError(t, err)
	assert.Equal(t, State{ID: "first"}, s, "the version is not settled")
}

func TestLoadWaitsForStateLock(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())

	// A writer holds the state lock while it replaces the file, which fails on
	// Windows while another process has the file open.
	unlock, err := lock(ctx, stateLockFileName, 0)
	require.NoError(t, err)
	done := make(chan struct{})
	go func() {
		_, err := Load(ctx)
		assert.NoError(t, err)
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("Load read the state while the state lock was held")
	case <-time.After(100 * time.Millisecond):
	}
	unlock()
	<-done
}

func TestLoadFailsWhenStateLockStaysHeld(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())

	// A stopped holder must not block every later command.
	unlock, err := lock(ctx, stateLockFileName, 0)
	require.NoError(t, err)
	defer unlock()
	start := time.Now()
	_, err = Load(ctx)
	assert.ErrorIs(t, err, errLockHeld)
	assert.Less(t, time.Since(start), stateLockWait+2*time.Second)
}

func TestLockReturnsOtherErrorsAtOnce(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())

	// A filesystem without locks fails every try, so it is not retried.
	errNoLocks := errors.New("no locks here")
	orig := tryLock
	t.Cleanup(func() { tryLock = orig })
	tries := 0
	tryLock = func(*os.File) error {
		tries++
		return errNoLocks
	}
	_, err := lock(ctx, stateLockFileName, stateLockWait)
	assert.ErrorIs(t, err, errNoLocks)
	assert.Equal(t, 1, tries)
}

func TestSetOptedOutWithoutLocks(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())

	// An explicit opt-out must work where the filesystem has no locks.
	orig := tryLock
	t.Cleanup(func() { tryLock = orig })
	tryLock = func(*os.File) error { return errors.New("no locks here") }
	require.NoError(t, SetOptedOut(ctx, true))

	s, err := load(ctx)
	require.NoError(t, err)
	assert.True(t, s.OptedOut)

	// A lock another process holds is waited for, not skipped.
	tryLock = func(*os.File) error { return errLockHeld }
	assert.ErrorIs(t, SetOptedOut(ctx, false), errLockHeld)
}

func TestRecordClearsInstallingWhenSettled(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())

	require.NoError(t, MarkInstalling(ctx))
	// A failed attempt leaves the marker for the next run.
	require.NoError(t, Record(ctx, "1.30.0", 1, "machine"))
	s, err := Load(ctx)
	require.NoError(t, err)
	assert.True(t, s.Installing)

	require.NoError(t, Record(ctx, "1.30.0", 0, "machine"))
	s, err = Load(ctx)
	require.NoError(t, err)
	assert.False(t, s.Installing)
}

func TestNextAttempt(t *testing.T) {
	assert.Equal(t, 1, NextAttempt(State{}, "1.30.0"))
	assert.Equal(t, 1, NextAttempt(State{LastVersion: "1.29.0", Attempts: 4}, "1.30.0"))
	assert.Equal(t, 5, NextAttempt(State{LastVersion: "1.30.0", Attempts: 4}, "1.30.0"))
}

func TestChildEnvCarriesTheRefusal(t *testing.T) {
	t.Cleanup(ResetDeclined)

	environ := []string{"HOME=/home/user"}
	assert.Equal(t, environ, ChildEnv(environ))

	// A CLI child started after a "no" must not install either, even when
	// the opt-out could not be stored.
	declined.Store(true)
	assert.Equal(t, []string{"HOME=/home/user", EnvVar + "=false"}, ChildEnv(environ))
}
