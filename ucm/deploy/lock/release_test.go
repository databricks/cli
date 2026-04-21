package lock_test

import (
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/ucm/deploy/lock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReleaseRoundTrip(t *testing.T) {
	ctx := t.Context()
	l := newTestLocker(t, "alice@example.com")

	require.NoError(t, l.Acquire(ctx, false))
	require.NoError(t, l.Release(ctx, lock.GoalDeploy))
	assert.False(t, l.Active)

	// Second Release errors since state is no longer Active.
	err := l.Release(ctx, lock.GoalDeploy)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unlock called when lock is not held")
}

func TestReleaseFailsWhenLockNotHeld(t *testing.T) {
	ctx := t.Context()
	l := newTestLocker(t, "alice@example.com")

	err := l.Release(ctx, lock.GoalDeploy)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unlock called when lock is not held")
}

func TestReleaseAfterContentionDoesNotClearOthersLock(t *testing.T) {
	ctx := t.Context()
	shared, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)

	first := newTestLockerOnFiler("alice@example.com", shared)
	second := newTestLockerOnFiler("bob@example.com", shared)

	require.NoError(t, first.Acquire(ctx, false))

	// second never acquired; its internal Active flag is false, so Release
	// must refuse to touch the on-disk lock.
	err = second.Release(ctx, lock.GoalDeploy)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unlock called when lock is not held")

	// first's lock is still on disk.
	active, err := first.GetActiveLockState(ctx)
	require.NoError(t, err)
	assert.Equal(t, first.LocalState.ID, active.ID)
}

func TestReleaseGoalDestroyToleratesMissingLockFile(t *testing.T) {
	ctx := t.Context()
	shared, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)

	l := newTestLockerOnFiler("alice@example.com", shared)
	require.NoError(t, l.Acquire(ctx, false))

	// Simulate the destroy race: some other component wiped the state dir
	// before we got a chance to Release.
	require.NoError(t, shared.Delete(ctx, lock.LockFileName))

	// Destroy must not fail.
	require.NoError(t, l.Release(ctx, lock.GoalDestroy))
	assert.False(t, l.Active)
}

func TestReleaseGoalDeployFailsOnMissingLockFile(t *testing.T) {
	ctx := t.Context()
	shared, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)

	l := newTestLockerOnFiler("alice@example.com", shared)
	require.NoError(t, l.Acquire(ctx, false))

	require.NoError(t, shared.Delete(ctx, lock.LockFileName))

	// GoalDeploy is strict: a missing lock file is an error state.
	err = l.Release(ctx, lock.GoalDeploy)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unlock called when lock is not held")
}

func TestReleaseWithAllowLockFileNotExistOption(t *testing.T) {
	ctx := t.Context()
	shared, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)

	l := newTestLockerOnFiler("alice@example.com", shared)
	require.NoError(t, l.Acquire(ctx, false))

	require.NoError(t, shared.Delete(ctx, lock.LockFileName))

	require.NoError(t, l.Release(ctx, lock.GoalDeploy, lock.AllowLockFileNotExist))
	assert.False(t, l.Active)
}
