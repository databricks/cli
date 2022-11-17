package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/databricks/bricks/folders"
	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"github.com/stretchr/testify/assert"
)

// TODO: create a utility function to create an empty test repo for tests before
// merging

func TestAccLock(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	// We assume cwd is in the bricks repo
	wd, err := os.Getwd()
	if err != nil {
		t.Log("[WARN] error fetching current working dir: ", err)
	}
	t.Log("test run dir: ", wd)
	bricksRepo, err := folders.FindDirWithLeaf(wd, ".git")
	if err != nil {
		t.Log("[ERROR] error finding git repo root in : ", wd)
	}
	t.Log("bricks repo location: : ", bricksRepo)
	assert.Equal(t, "bricks", filepath.Base(bricksRepo))

	wsc := workspaces.New()
	ctx := context.Background()
	me, err := wsc.CurrentUser.Me(ctx)
	assert.NoError(t, err)
	repoUrl := "https://github.com/shreyas-goenka/empty-repo.git"
	repoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, RandomName("empty-repo-lock-integration-"))

	repoInfo, err := wsc.Repos.Create(ctx, repos.CreateRepo{
		Path:     repoPath,
		Url:      repoUrl,
		Provider: "gitHub",
	})
	assert.NoError(t, err)

	t.Cleanup(func() {
		err := wsc.Repos.DeleteByRepoId(ctx, repoInfo.Id)
		assert.NoError(t, err)
	})

	// clone public empty remote repo
	tempDir := t.TempDir()
	cmd := exec.Command("git", "clone", repoUrl)
	cmd.Dir = tempDir
	err = cmd.Run()
	assert.NoError(t, err)
}
