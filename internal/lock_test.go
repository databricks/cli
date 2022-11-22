package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/databricks/bricks/cmd/deploy"
	"github.com/databricks/bricks/project"
	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/stretchr/testify/assert"
)

// TODO: create a utility function to create an empty test repo for tests before
// merging

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

func printAllFiles(t *testing.T, ctx context.Context, prefix string) {
	prj := project.Get(ctx)
	all, err := prj.ListRemoteFiles(ctx)
	assert.NoError(t, err)
	t.Logf("%s: %v", prefix, all)
}

func TestAccLock(t *testing.T) {
	// t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))
	ctx := createLocalTestProject(t)
	remoteProjectRoot := createRemoteTestProject(t, ctx, "lock-acc-")
	locker, err := deploy.CreateLocker(ctx, false, remoteProjectRoot)
	assert.NoError(t, err)
	err = locker.Lock(ctx)
	assert.NoError(t, err)
	err = locker.Unlock(ctx)
	assert.NoError(t, err)
	err = locker.Lock(ctx)
	assert.NoError(t, err)
	assert.True(t, false)
}
