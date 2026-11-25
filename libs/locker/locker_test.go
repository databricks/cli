package locker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eofFiler fails the first `failures` writes with the error net/http returns when the connection
// is cut mid-response. The write numbered appliedOn (1-based, 0 for none) still writes the file
// before failing, like a request the server applied before the connection was cut.
type eofFiler struct {
	filer.Filer
	failures  int
	appliedOn int
	writes    int
}

func (f *eofFiler) Write(ctx context.Context, path string, reader io.Reader, mode ...filer.WriteMode) error {
	f.writes++
	if f.writes > f.failures {
		return f.Filer.Write(ctx, path, reader, mode...)
	}
	if f.writes == f.appliedOn {
		if err := f.Filer.Write(ctx, path, reader, mode...); err != nil {
			return err
		}
	}
	return &url.Error{Op: "Post", URL: "https://example.com/" + path, Err: io.ErrUnexpectedEOF}
}

func newTestLocker(t *testing.T, f filer.Filer) *Locker {
	return &Locker{
		filer:     f,
		TargetDir: t.TempDir(),
		State:     &LockState{ID: uuid.New(), User: "tester"},
	}
}

func TestLockRetriesUnexpectedEOF(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")

	tests := []struct {
		name      string
		failures  int
		appliedOn int
	}{
		// The retry creates the lock.
		{name: "not applied", failures: 1, appliedOn: 0},
		// The retry fails with "already exists" and the read-back finds our lock.
		{name: "applied", failures: 1, appliedOn: 1},
		// Retries run out, but the last attempt was applied: the read-back finds our lock.
		{name: "last attempt applied", failures: 3, appliedOn: 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			local, err := filer.NewLocalClient(t.TempDir())
			require.NoError(t, err)
			f := &eofFiler{Filer: local, failures: tc.failures, appliedOn: tc.appliedOn}
			locker := newTestLocker(t, f)

			require.NoError(t, locker.Lock(ctx, false))
			assert.True(t, locker.Active)
			assert.Equal(t, min(tc.failures+1, 3), f.writes)
		})
	}
}

func TestLockFailsWhenEveryAttemptHitsUnexpectedEOF(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	local, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	f := &eofFiler{Filer: local, failures: 3}
	locker := newTestLocker(t, f)

	err = locker.Lock(ctx, false)
	require.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.False(t, locker.Active)
	assert.Equal(t, 3, f.writes)
}

// lateFiler fails the first `failures` writes with the SDK's inactivity timeout without applying
// them. applyLate then applies the first of them, as the server does with a request it was still
// processing.
type lateFiler struct {
	filer.Filer
	failures int
	writes   int
	stalled  []byte
}

func (f *lateFiler) Write(ctx context.Context, path string, reader io.Reader, mode ...filer.WriteMode) error {
	f.writes++
	if f.writes > f.failures {
		return f.Filer.Write(ctx, path, reader, mode...)
	}
	if f.stalled == nil {
		body, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		f.stalled = body
	}
	return &url.Error{Op: "Post", URL: "https://example.com/" + path, Err: errors.New("request timed out after 1m30s of inactivity")}
}

func (f *lateFiler) applyLate(t *testing.T) {
	require.NoError(t, f.Filer.Write(t.Context(), LockFileName, bytes.NewReader(f.stalled)))
}

func TestLockReplacesLockReleasedBeforeLateWrite(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	local, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	released := filepath.Join(t.TempDir(), "released.json")

	f := &lateFiler{Filer: local, failures: 1}
	first := newTestLocker(t, f)
	first.ReleasedLocksPath = released
	require.NoError(t, first.Lock(ctx, false))
	require.NoError(t, first.Unlock(ctx))
	f.applyLate(t)

	second := newTestLocker(t, local)
	second.ReleasedLocksPath = released
	require.NoError(t, second.Lock(ctx, false))
	state, err := second.GetActiveLockState(ctx)
	require.NoError(t, err)
	assert.Equal(t, second.State.ID, state.ID)
}

func TestLockReplacesLockLeftByFailedLock(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	local, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	released := filepath.Join(t.TempDir(), "released.json")

	// Every attempt times out; the first is applied after Lock gave up.
	f := &lateFiler{Filer: local, failures: 3}
	first := newTestLocker(t, f)
	first.ReleasedLocksPath = released
	require.Error(t, first.Lock(ctx, false))
	f.applyLate(t)

	second := newTestLocker(t, local)
	second.ReleasedLocksPath = released
	require.NoError(t, second.Lock(ctx, false))
}

func TestLockKeepsLockNotReleasedHere(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	local, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	released := filepath.Join(t.TempDir(), "released.json")

	// A lock released after an uncertain write is recorded, but a different, live lock is kept.
	f := &lateFiler{Filer: local, failures: 1}
	first := newTestLocker(t, f)
	first.ReleasedLocksPath = released
	require.NoError(t, first.Lock(ctx, false))
	require.NoError(t, first.Unlock(ctx))

	other := newTestLocker(t, local)
	require.NoError(t, other.Lock(ctx, false))

	second := newTestLocker(t, local)
	second.ReleasedLocksPath = released
	err = second.Lock(ctx, false)
	require.ErrorContains(t, err, "deploy lock acquired by tester")
	assert.True(t, other.Active)
	state, err := second.GetActiveLockState(ctx)
	require.NoError(t, err)
	assert.Equal(t, other.State.ID, state.ID)
}

// deleteFailingFiler fails every delete without applying it, as when the request times out.
type deleteFailingFiler struct {
	filer.Filer
	deletes int

	// onDelete runs on every delete, e.g. to move the clock as a slow, retried delete does.
	onDelete func()
}

func (f *deleteFailingFiler) Delete(ctx context.Context, path string, mode ...filer.DeleteMode) error {
	f.deletes++
	if f.onDelete != nil {
		f.onDelete()
	}
	return &url.Error{Op: "Post", URL: "https://example.com/" + path, Err: errors.New("request timed out after 1m30s of inactivity")}
}

func TestLockReplacesLockLeftByFailedUnlock(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	local, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	released := filepath.Join(t.TempDir(), "released.json")
	now := time.Now()
	timeNow = func() time.Time { return now }
	t.Cleanup(func() { timeNow = time.Now })

	first := newTestLocker(t, &deleteFailingFiler{Filer: local})
	first.ReleasedLocksPath = released
	require.NoError(t, first.Lock(ctx, false))
	require.Error(t, first.Unlock(ctx))

	// The failed delete may still land, so the lock is kept while it can: replacing it now would
	// let that delete remove the new lock.
	f := &deleteFailingFiler{Filer: local}
	second := newTestLocker(t, f)
	second.ReleasedLocksPath = released
	require.ErrorContains(t, second.Lock(ctx, false), "deploy lock acquired by tester")
	assert.Equal(t, 0, f.deletes)

	now = now.Add(pendingDeleteWindow)
	third := newTestLocker(t, local)
	third.ReleasedLocksPath = released
	require.NoError(t, third.Lock(ctx, false))
	state, err := third.GetActiveLockState(ctx)
	require.NoError(t, err)
	assert.Equal(t, third.State.ID, state.ID)
}

// deleteThenTimeoutFiler applies the first delete but answers it with a timeout, and runs
// afterDelete in between, as when another deployer takes the lock before the retry.
type deleteThenTimeoutFiler struct {
	filer.Filer
	afterDelete func()
	deletes     int
}

func (f *deleteThenTimeoutFiler) Delete(ctx context.Context, path string, mode ...filer.DeleteMode) error {
	f.deletes++
	err := f.Filer.Delete(ctx, path, mode...)
	if f.deletes > 1 || err != nil {
		return err
	}
	f.afterDelete()
	return &url.Error{Op: "Post", URL: "https://example.com/" + path, Err: errors.New("request timed out after 1m30s of inactivity")}
}

func TestStaleLockRemovalKeepsLockTakenBeforeRetry(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	local, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	released := filepath.Join(t.TempDir(), "released.json")

	leaveStaleLock(t, ctx, local, released)

	other := newTestLocker(t, local)
	f := &deleteThenTimeoutFiler{Filer: local, afterDelete: func() {
		require.NoError(t, other.Lock(ctx, false))
	}}
	second := newTestLocker(t, retry.Filer(f))
	second.ReleasedLocksPath = released
	err = second.Lock(ctx, false)
	require.ErrorContains(t, err, "deploy lock acquired by tester")
	assert.Equal(t, 1, f.deletes)
	state, err := second.GetActiveLockState(ctx)
	require.NoError(t, err)
	assert.Equal(t, other.State.ID, state.ID)
}

func TestStaleLockRemovalSendsOneDelete(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	local, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	released := filepath.Join(t.TempDir(), "released.json")

	leaveStaleLock(t, ctx, local, released)

	// The delete times out and may land later, so it is not sent again: a second delete could
	// remove a lock taken after the first one landed. The lock stays and Lock names its holder.
	f := &deleteFailingFiler{Filer: local}
	second := newTestLocker(t, retry.Filer(f))
	second.ReleasedLocksPath = released
	err = second.Lock(ctx, false)
	require.ErrorContains(t, err, "deploy lock acquired by tester")
	assert.Equal(t, 1, f.deletes)
}

func TestStaleLockRemovalRemembersPendingDelete(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	local, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	released := filepath.Join(t.TempDir(), "released.json")
	now := time.Now()
	timeNow = func() time.Time { return now }
	t.Cleanup(func() { timeNow = time.Now })

	leaveStaleLock(t, ctx, local, released)

	// The next command's delete times out and may land later.
	second := newTestLocker(t, &deleteFailingFiler{Filer: local})
	second.ReleasedLocksPath = released
	require.ErrorContains(t, second.Lock(ctx, false), "deploy lock acquired by tester")

	// A later command does not delete the lock again while that delete may still land: it would
	// remove the lock this command then creates.
	f := &deleteFailingFiler{Filer: local}
	third := newTestLocker(t, f)
	third.ReleasedLocksPath = released
	require.ErrorContains(t, third.Lock(ctx, false), "deploy lock acquired by tester")
	assert.Equal(t, 0, f.deletes)

	// Once the window has passed, the stale lock is removed.
	now = now.Add(pendingDeleteWindow)
	fourth := newTestLocker(t, local)
	fourth.ReleasedLocksPath = released
	require.NoError(t, fourth.Lock(ctx, false))
}

// leaveStaleLock takes and releases the lock, then writes it back, as a late lock write does.
func leaveStaleLock(t *testing.T, ctx context.Context, local filer.Filer, released string) {
	stale := newTestLocker(t, local)
	stale.ReleasedLocksPath = released
	require.NoError(t, stale.Lock(ctx, false))
	data, err := json.Marshal(stale.State)
	require.NoError(t, err)
	require.NoError(t, stale.Unlock(ctx))
	require.NoError(t, local.Write(ctx, LockFileName, bytes.NewReader(data)))
}

func TestPendingDeleteWindowStartsWhenUnlockDeleteFails(t *testing.T) {
	ctx := env.Set(t.Context(), retry.IntervalMsVariable, "1")
	local, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	released := filepath.Join(t.TempDir(), "released.json")
	now := time.Now()
	timeNow = func() time.Time { return now }
	t.Cleanup(func() { timeNow = time.Now })

	// The SDK can resend the delete for longer than the window before it fails.
	first := newTestLocker(t, &deleteFailingFiler{Filer: local, onDelete: func() { now = now.Add(pendingDeleteWindow) }})
	first.ReleasedLocksPath = released
	require.NoError(t, first.Lock(ctx, false))
	require.Error(t, first.Unlock(ctx))

	now = now.Add(pendingDeleteWindow - time.Minute)
	f := &deleteFailingFiler{Filer: local}
	second := newTestLocker(t, f)
	second.ReleasedLocksPath = released
	require.ErrorContains(t, second.Lock(ctx, false), "deploy lock acquired by tester")
	assert.Equal(t, 0, f.deletes)
}
