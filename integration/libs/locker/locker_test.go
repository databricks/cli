package locker_test

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/libs/filer"
	lockpkg "github.com/databricks/cli/libs/locker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLock(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	wsc := wt.W
	remoteProjectRoot := acc.TemporaryWorkspaceDir(wt, "lock-acc-")

	// 5 lockers try to acquire a lock at the same time
	numConcurrentLocks := 5

	// Keep single locker unlocked.
	// We use this to check on the current lock through GetActiveLockState.
	locker, err := lockpkg.CreateLocker("humpty.dumpty@databricks.com", remoteProjectRoot, wsc)
	require.NoError(t, err)

	lockerErrs := make([]error, numConcurrentLocks)
	lockers := make([]*lockpkg.Locker, numConcurrentLocks)
	for i := range numConcurrentLocks {
		lockers[i], err = lockpkg.CreateLocker("humpty.dumpty@databricks.com", remoteProjectRoot, wsc)
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	for currentIndex := range numConcurrentLocks {
		wg.Add(1)
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			lockerErrs[currentIndex] = lockers[currentIndex].Lock(ctx, false)
		}()
	}
	wg.Wait()

	countActive := 0
	indexOfActiveLocker := 0
	indexOfAnInactiveLocker := -1
	for i := range numConcurrentLocks {
		if lockers[i].Active {
			countActive += 1
			assert.NoError(t, lockerErrs[i])
			indexOfActiveLocker = i
		} else {
			if indexOfAnInactiveLocker == -1 {
				indexOfAnInactiveLocker = i
			}
			assert.ErrorContains(t, lockerErrs[i], "lock acquired by")
			assert.ErrorContains(t, lockerErrs[i], "Use --force-lock to override")
		}
	}
	assert.Equal(t, 1, countActive, "Exactly one locker should successfull acquire the lock")

	// test remote lock matches active lock
	remoteLocker, err := locker.GetActiveLockState(ctx)
	require.NoError(t, err)
	assert.Equal(t, remoteLocker.ID, lockers[indexOfActiveLocker].State.ID, "remote locker id does not match active locker")
	assert.True(t, remoteLocker.AcquisitionTime.Equal(lockers[indexOfActiveLocker].State.AcquisitionTime), "remote locker acquisition time does not match active locker")

	// test all other locks (inactive ones) do not match the remote lock and Unlock fails
	for i := range numConcurrentLocks {
		if i == indexOfActiveLocker {
			continue
		}
		assert.NotEqual(t, remoteLocker.ID, lockers[i].State.ID)
		err := lockers[i].Unlock(ctx)
		assert.ErrorContains(t, err, "unlock called when lock is not held")
	}

	// test inactive locks fail to write a file
	for i := range numConcurrentLocks {
		if i == indexOfActiveLocker {
			continue
		}
		err := lockers[i].Write(ctx, "foo.json", []byte(`'{"surname":"Khan", "name":"Shah Rukh"}`))
		assert.ErrorContains(t, err, "failed to put file. deploy lock not held")
	}

	// active locker file write succeeds
	err = lockers[indexOfActiveLocker].Write(ctx, "foo.json", []byte(`{"surname":"Khan", "name":"Shah Rukh"}`))
	assert.NoError(t, err)

	// read active locker file
	r, err := lockers[indexOfActiveLocker].Read(ctx, "foo.json")
	require.NoError(t, err)
	defer r.Close()
	b, err := io.ReadAll(r)
	require.NoError(t, err)

	// assert on active locker content
	var res map[string]string
	err = json.Unmarshal(b, &res)
	require.NoError(t, err)
	assert.NoError(t, err)
	assert.Equal(t, "Khan", res["surname"])
	assert.Equal(t, "Shah Rukh", res["name"])

	// inactive locker file reads fail
	for i := range numConcurrentLocks {
		if i == indexOfActiveLocker {
			continue
		}
		_, err = lockers[i].Read(ctx, "foo.json")
		assert.ErrorContains(t, err, "failed to get file. deploy lock not held")
	}

	// Unlock active lock and check it becomes inactive
	err = lockers[indexOfActiveLocker].Unlock(ctx)
	assert.NoError(t, err)
	remoteLocker, err = locker.GetActiveLockState(ctx)
	assert.ErrorIs(t, err, fs.ErrNotExist, "remote lock file not deleted on unlock")
	assert.Nil(t, remoteLocker)
	assert.False(t, lockers[indexOfActiveLocker].Active)

	// A locker that failed to acquire the lock should now be able to acquire it
	assert.False(t, lockers[indexOfAnInactiveLocker].Active)
	err = lockers[indexOfAnInactiveLocker].Lock(ctx, false)
	assert.NoError(t, err)
	assert.True(t, lockers[indexOfAnInactiveLocker].Active)
}

func setupLockerTest(t *testing.T) (context.Context, *lockpkg.Locker, filer.Filer) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	// create temp wsfs dir
	tmpDir := acc.TemporaryWorkspaceDir(wt, "locker-")
	f, err := filer.NewWorkspaceFilesClient(w, tmpDir)
	require.NoError(t, err)

	// create locker
	locker, err := lockpkg.CreateLocker("redfoo@databricks.com", tmpDir, w)
	require.NoError(t, err)

	return ctx, locker, f
}

func TestLockUnlockWithoutAllowsLockFileNotExist(t *testing.T) {
	ctx, locker, f := setupLockerTest(t)
	var err error

	// Acquire lock on tmp directory
	err = locker.Lock(ctx, false)
	require.NoError(t, err)

	// Assert lock file is created
	_, err = f.Stat(ctx, "deploy.lock")
	assert.NoError(t, err)

	// Manually delete lock file
	err = f.Delete(ctx, "deploy.lock")
	assert.NoError(t, err)

	// Assert error, because lock file does not exist
	err = locker.Unlock(ctx)
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestLockUnlockWithAllowsLockFileNotExist(t *testing.T) {
	ctx, locker, f := setupLockerTest(t)
	var err error

	// Acquire lock on tmp directory
	err = locker.Lock(ctx, false)
	require.NoError(t, err)
	assert.True(t, locker.Active)

	// Assert lock file is created
	_, err = f.Stat(ctx, "deploy.lock")
	assert.NoError(t, err)

	// Manually delete lock file
	err = f.Delete(ctx, "deploy.lock")
	assert.NoError(t, err)

	// Assert error, because lock file does not exist
	err = locker.Unlock(ctx, lockpkg.AllowLockFileNotExist)
	assert.NoError(t, err)
	assert.False(t, locker.Active)
}
