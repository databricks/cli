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
	wsc := workspaces.New()
	ctx := context.Background()
	me, err := wsc.CurrentUser.Me(ctx)
	assert.NoError(t, err)
	repoUrl := "https://github.com/shreyas-goenka/empty-repo.git"
	repoPath := fmt.Sprintf("/Repos/%s/empty-repo", me.UserName)

	repoInfo, err := wsc.Repos.Create(ctx, repos.CreateRepo{
		Path:     repoPath,
		Url:      repoUrl,
		Provider: "gitHub",
	})
	assert.NoError(t, err)

	t.Cleanup(func() {
		err := wsc.Repos.DeleteByRepoId(ctx, fmt.Sprint(repoInfo.Id))
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
	cmd = exec.Command("bricks", "sync")
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
	var files []string
	for _, v := range repoContent.Objects {
		files = append(files, filepath.Base(v.Path))
	}
	assert.Contains(t, files, "databricks.yml")
	assert.Contains(t, files, ".gitkeep")

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
	for _, v := range repoContent.Objects {
		files = append(files, filepath.Base(v.Path))
	}
	assert.Contains(t, files, "databricks.yml")
	assert.Contains(t, files, ".gitkeep")
	assert.Contains(t, files, "hello.txt")
	assert.Contains(t, files, "world.txt")

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
	for _, v := range repoContent.Objects {
		files = append(files, filepath.Base(v.Path))
	}
	assert.Contains(t, files, "databricks.yml")
	assert.Contains(t, files, ".gitkeep")
	assert.Contains(t, files, "world.txt")
}
