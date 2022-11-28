package internal

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"github.com/stretchr/testify/assert"
)

// TODO: create a utility function to create an empty test repo for tests and refactor sync_test integration test

const EmptyRepoUrl = "https://github.com/shreyas-goenka/empty-repo.git"

func createRemoteTestProject(t *testing.T, projectNamePrefix string, wsc *workspaces.WorkspacesClient) string {
	ctx := context.TODO()
	me, err := wsc.CurrentUser.Me(ctx)
	assert.NoError(t, err)

	remoteProjectRoot := fmt.Sprintf("/Repos/%s/%s", me.UserName, RandomName(projectNamePrefix))
	repoInfo, err := wsc.Repos.Create(ctx, repos.CreateRepo{
		Path:     remoteProjectRoot,
		Url:      EmptyRepoUrl,
		Provider: "gitHub",
	})
	assert.NoError(t, err)
	t.Cleanup(func() {
		err := wsc.Repos.DeleteByRepoId(ctx, repoInfo.Id)
		assert.NoError(t, err)
	})

	return remoteProjectRoot
}

func createLocalTestProject(t *testing.T) string {
	tempDir := t.TempDir()

	cmd := exec.Command("git", "clone", EmptyRepoUrl)
	cmd.Dir = tempDir
	err := cmd.Run()
	assert.NoError(t, err)

	localProjectRoot := filepath.Join(tempDir, "empty-repo")
	err = os.Chdir(localProjectRoot)
	assert.NoError(t, err)
	return localProjectRoot
}

func TestAccLock(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))
	ctx := context.TODO()
	wsc := workspaces.New()
	createLocalTestProject(t)
	remoteProjectRoot := createRemoteTestProject(t, "lock-acc-", wsc)

	// 50 lockers try to acquire a lock at the same time
	numConcurrentLocks := 50

	var err error
	lockerErrs := make([]error, numConcurrentLocks)
	lockers := make([]*bundle.DeployLocker, numConcurrentLocks)

	for i := 0; i < numConcurrentLocks; i++ {
		lockers[i], err = lock.CreateLocker("humpty.dumpty@databricks.com", false, remoteProjectRoot)
		assert.NoError(t, err)
	}

	var wg sync.WaitGroup
	for i := 0; i < numConcurrentLocks; i++ {
		wg.Add(1)
		currentIndex := i
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			lockerErrs[currentIndex] = lockers[currentIndex].Lock(ctx, wsc)
		}()
	}
	wg.Wait()

	countActive := 0
	indexOfActiveLocker := 0
	indexOfAnInactiveLocker := -1
	for i := 0; i < numConcurrentLocks; i++ {
		if lockers[i].Active {
			countActive += 1
			assert.NoError(t, lockerErrs[i])
			indexOfActiveLocker = i
		} else {
			if indexOfAnInactiveLocker == -1 {
				indexOfAnInactiveLocker = i
			}
			assert.ErrorContains(t, lockerErrs[i], "ongoing deployment")
			assert.ErrorContains(t, lockerErrs[i], "Use --force to forcibly deploy your bundle")
		}
	}
	assert.Equal(t, 1, countActive, "Exactly one locker should successfull acquire the lock")

	// test remote lock matches active lock
	remoteLocker, err := lock.GetRemoteLocker(ctx, wsc, lockers[indexOfActiveLocker].RemotePath())
	assert.NoError(t, err)
	assert.Equal(t, remoteLocker.Id, lockers[indexOfActiveLocker].Id, "remote locker id does not match active locker")
	assert.True(t, remoteLocker.AcquisitionTime.Equal(lockers[indexOfActiveLocker].AcquisitionTime), "remote locker acquisition time does not match active locker")

	// test all other locks (inactive ones) do not match the remote lock and Unlock fails
	for i := 0; i < numConcurrentLocks; i++ {
		if i == indexOfActiveLocker {
			continue
		}
		assert.NotEqual(t, remoteLocker.Id, lockers[i].Id)
		err := lockers[i].Unlock(ctx, wsc)
		assert.ErrorContains(t, err, "only active lockers can be unlocked")
	}

	// Unlock active lock and check it becomes inactive
	err = lockers[indexOfActiveLocker].Unlock(ctx, wsc)
	assert.NoError(t, err)
	remoteLocker, err = lock.GetRemoteLocker(ctx, wsc, lockers[indexOfActiveLocker].RemotePath())
	assert.ErrorContains(t, err, "File not found.", "remote lock file not deleted on unlock")
	assert.Nil(t, remoteLocker)
	assert.False(t, lockers[indexOfActiveLocker].Active)

	// A locker that failed to acquire the lock should now be able to acquire it
	assert.False(t, lockers[indexOfAnInactiveLocker].Active)
	err = lockers[indexOfAnInactiveLocker].Lock(ctx, wsc)
	assert.NoError(t, err)
	assert.True(t, lockers[indexOfAnInactiveLocker].Active)
}
