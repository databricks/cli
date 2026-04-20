package lock_test

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLocker(t *testing.T, user string) *lock.Locker {
	t.Helper()
	f, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	return lock.NewLockerWithFiler(user, "/test", f)
}

func newTestLockerOnFiler(user string, f filer.Filer) *lock.Locker {
	return lock.NewLockerWithFiler(user, "/test", f)
}

func TestAcquireSucceedsOnEmptyDir(t *testing.T) {
	ctx := t.Context()
	l := newTestLocker(t, "alice@example.com")

	err := l.Acquire(ctx, false)
	require.NoError(t, err)
	assert.True(t, l.Active)
	assert.Equal(t, "alice@example.com", l.LocalState.User)
	assert.False(t, l.LocalState.IsForced)
	assert.False(t, l.LocalState.AcquisitionTime.IsZero())
}

func TestAcquireWritesRecordReadableByGetActiveLockState(t *testing.T) {
	ctx := t.Context()
	l := newTestLocker(t, "alice@example.com")

	require.NoError(t, l.Acquire(ctx, false))

	active, err := l.GetActiveLockState(ctx)
	require.NoError(t, err)
	assert.Equal(t, l.LocalState.ID, active.ID)
	assert.Equal(t, "alice@example.com", active.User)
}

func TestAcquireContentionReturnsErrLockHeld(t *testing.T) {
	ctx := t.Context()

	// Two lockers share the same underlying filer — i.e. same remote
	// state dir. The second Acquire must observe the first's record.
	shared, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)

	first := newTestLockerOnFiler("alice@example.com", shared)
	second := newTestLockerOnFiler("bob@example.com", shared)

	require.NoError(t, first.Acquire(ctx, false))

	err = second.Acquire(ctx, false)
	require.Error(t, err)

	var held *lock.ErrLockHeld
	require.ErrorAs(t, err, &held)
	assert.Equal(t, "alice@example.com", held.Holder)
	assert.False(t, held.IsForced)
	assert.False(t, second.Active)
}

func TestAcquireForceOverridesExistingLock(t *testing.T) {
	ctx := t.Context()
	shared, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)

	first := newTestLockerOnFiler("alice@example.com", shared)
	second := newTestLockerOnFiler("bob@example.com", shared)

	require.NoError(t, first.Acquire(ctx, false))

	// Non-forced contention fails.
	err = second.Acquire(ctx, false)
	require.Error(t, err)
	var held *lock.ErrLockHeld
	require.ErrorAs(t, err, &held)

	// Forced contention wins and flips the active record over to second.
	require.NoError(t, second.Acquire(ctx, true))
	assert.True(t, second.Active)
	assert.True(t, second.LocalState.IsForced)

	active, err := second.GetActiveLockState(ctx)
	require.NoError(t, err)
	assert.Equal(t, second.LocalState.ID, active.ID)
	assert.Equal(t, "bob@example.com", active.User)
}

func TestErrLockHeldMessageFormat(t *testing.T) {
	err := &lock.ErrLockHeld{Holder: "alice@example.com"}
	assert.Contains(t, err.Error(), "deploy lock acquired by alice@example.com")
	assert.Contains(t, err.Error(), "Use --force-lock to override")

	forcedErr := &lock.ErrLockHeld{Holder: "alice@example.com", IsForced: true}
	assert.Contains(t, forcedErr.Error(), "deploy lock force acquired by alice@example.com")
}

func TestErrLockHeldIsErrorAsTargetable(t *testing.T) {
	var zero *lock.ErrLockHeld
	err := error(&lock.ErrLockHeld{Holder: "alice@example.com"})
	assert.True(t, errors.As(err, &zero))
}
