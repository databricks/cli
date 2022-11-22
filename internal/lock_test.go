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

	"github.com/databricks/bricks/cmd/deploy"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/stretchr/testify/assert"
)

// TODO: create a utility function to create an empty test repo for tests and refactor sync_test integration test

const EmptyRepoUrl = "https://github.com/shreyas-goenka/empty-repo.git"

func createRemoteTestProject(t *testing.T, ctx context.Context, projectNamePrefix string) string {
	prj := project.Get(ctx)
	wsc := prj.WorkspacesClient()
	me, err := wsc.CurrentUser.Me(ctx)
	assert.NoError(t, err)

	remoteProjectRoot := fmt.Sprintf("/Repos/%s/%s", me.UserName, RandomName(projectNamePrefix))
	repoInfo, err := wsc.Repos.Create(ctx, repos.CreateRepo{
		Path:     remoteProjectRoot,
		Url:      EmptyRepoUrl,
		Provider: "gitHub",
	})
	prj.OverrideRemoteRoot(remoteProjectRoot)
	assert.NoError(t, err)
	t.Cleanup(func() {
		err := wsc.Repos.DeleteByRepoId(ctx, repoInfo.Id)
		assert.NoError(t, err)
	})

	return remoteProjectRoot
}

func createLocalTestProject(t *testing.T) context.Context {
	ctx := context.Background()
	tempDir := t.TempDir()

	cmd := exec.Command("git", "clone", EmptyRepoUrl)
	cmd.Dir = tempDir
	err := cmd.Run()
	assert.NoError(t, err)

	localProjectRoot := filepath.Join(tempDir, "empty-repo")
	err = os.Chdir(localProjectRoot)
	assert.NoError(t, err)
	ctx, err = project.Initialize(ctx, localProjectRoot, project.DefaultEnvironment)
	assert.NoError(t, err)
	return ctx
}

func TestAccLock(t *testing.T) {
	// t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))
	ctx := createLocalTestProject(t)
	remoteProjectRoot := createRemoteTestProject(t, ctx, "lock-acc-")
	numConcurrentLocks := 50

	var err error
	lockerErrs := make([]error, numConcurrentLocks)
	lockers := make([]*deploy.DeployLocker, numConcurrentLocks)

	for i := 0; i < numConcurrentLocks; i++ {
		lockers[i], err = deploy.CreateLocker(ctx, false, remoteProjectRoot)
		assert.NoError(t, err)
	}

	// 50 lockers try to acquire a lock at the same time
	var wg sync.WaitGroup
	for i := 0; i < numConcurrentLocks; i++ {
		wg.Add(1)
		currentIndex := i
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
			lockerErrs[currentIndex] = lockers[currentIndex].Lock(ctx)
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
			assert.ErrorContains(t, lockerErrs[i], "cannot deploy")
			assert.ErrorContains(t, lockerErrs[i], "Use --force to forcibly deploy your bundle")
		}
	}
	assert.Equal(t, 1, countActive, "Exactly one locker should acquire the lock")

	// test remote lock matches active lock
	remoteLocker, err := deploy.GetRemoteLocker(ctx, filepath.Join(lockers[indexOfActiveLocker].TargetDir, ".bundle/deploy.lock"))
	assert.NoError(t, err)
	assert.Equal(t, remoteLocker.Id, lockers[indexOfActiveLocker].Id, "remote locker id does not match active locker")
	assert.True(t, remoteLocker.AcquisitionTime.Equal(lockers[indexOfActiveLocker].AcquisitionTime), "remote locker acquisition time does not match active locker")

	// test all other locks (inactive ones) do not match the remote lock and Unlock fails
	for i := 0; i < numConcurrentLocks; i++ {
		if i == indexOfActiveLocker {
			continue
		}
		assert.NotEqual(t, remoteLocker.Id, lockers[i].Id)
		err := lockers[i].Unlock(ctx)
		assert.ErrorContains(t, err, "only active lockers can be unlocked")
	}

	// Unlock active lock and check it becomes inactive
	err = lockers[indexOfActiveLocker].Unlock(ctx)
	assert.NoError(t, err)
	remoteLocker, err = deploy.GetRemoteLocker(ctx, filepath.Join(lockers[indexOfActiveLocker].TargetDir, ".bundle/deploy.lock"))
	assert.ErrorContains(t, err, "File not found.", "remote lock file not deleted on unlock")
	assert.Nil(t, remoteLocker)
	assert.False(t, lockers[indexOfActiveLocker].Active)

	// A locker that failed to acquire the lock should now be able to acquire it
	assert.False(t, lockers[indexOfAnInactiveLocker].Active)
	err = lockers[indexOfAnInactiveLocker].Lock(ctx)
	assert.NoError(t, err)
	assert.True(t, lockers[indexOfAnInactiveLocker].Active)
}
