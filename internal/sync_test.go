package internal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"github.com/stretchr/testify/assert"
)

func TestAccSync(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := workspaces.New()
	ctx := context.Background()
	me, err := wsc.CurrentUser.Me(ctx)
	assert.NoError(t, err)
	repoUrl := "https://github.com/shreyas-goenka/empty-repo.git"
	repoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, RandomName("empty-repo-sync-integration-"))

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

	// clone public remote repo
	tempDir := t.TempDir()
	cmd := exec.Command("git", "clone", repoUrl)
	cmd.Dir = tempDir
	err = cmd.Run()
	assert.NoError(t, err)

	// Initialize the databrick.yml config
	projectDir := filepath.Join(tempDir, "empty-repo")
	content := []byte("name: test-project\nprofile: DEFAULT")
	f, err := os.Create(filepath.Join(projectDir, "databricks.yml"))
	assert.NoError(t, err)
	defer f.Close()
	_, err = f.Write(content)
	assert.NoError(t, err)

	// start bricks sync process
	cmd = exec.Command("bricks", "sync", "--remote-path", repoPath)
	cmd.Dir = projectDir
	err = cmd.Start()
	assert.NoError(t, err)
	t.Cleanup(func() {
		cmd.Process.Kill()
	})

	// First upload assertion
	assert.Eventually(t, func() bool {
		repoContent, err := wsc.Workspace.List(ctx, workspace.ListRequest{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(repoContent.Objects) == 2
	}, 30*time.Second, time.Second)
	repoContent, err := wsc.Workspace.List(ctx, workspace.ListRequest{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files1 []string
	for _, v := range repoContent.Objects {
		files1 = append(files1, filepath.Base(v.Path))
	}
	assert.Len(t, files1, 2)
	assert.Contains(t, files1, "databricks.yml")
	assert.Contains(t, files1, ".gitkeep")

	// Create new files and assert
	os.Create(filepath.Join(projectDir, "hello.txt"))
	os.Create(filepath.Join(projectDir, "world.txt"))
	assert.Eventually(t, func() bool {
		repoContent, err := wsc.Workspace.List(ctx, workspace.ListRequest{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(repoContent.Objects) == 4
	}, 30*time.Second, time.Second)
	repoContent, err = wsc.Workspace.List(ctx, workspace.ListRequest{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files2 []string
	for _, v := range repoContent.Objects {
		files2 = append(files2, filepath.Base(v.Path))
	}
	assert.Len(t, files2, 4)
	assert.Contains(t, files2, "databricks.yml")
	assert.Contains(t, files2, ".gitkeep")
	assert.Contains(t, files2, "hello.txt")
	assert.Contains(t, files2, "world.txt")

	// delete a file and assert
	os.Remove(filepath.Join(projectDir, "hello.txt"))
	assert.Eventually(t, func() bool {
		repoContent, err := wsc.Workspace.List(ctx, workspace.ListRequest{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(repoContent.Objects) == 3
	}, 30*time.Second, time.Second)
	repoContent, err = wsc.Workspace.List(ctx, workspace.ListRequest{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files3 []string
	for _, v := range repoContent.Objects {
		files3 = append(files3, filepath.Base(v.Path))
	}
	assert.Len(t, files3, 3)
	assert.Contains(t, files3, "databricks.yml")
	assert.Contains(t, files3, ".gitkeep")
	assert.Contains(t, files3, "world.txt")
}
