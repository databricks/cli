package internal

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/bricks/folders"
	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/databricks/databricks-sdk-go/workspaces"
	"github.com/stretchr/testify/assert"
)

// TODO: Write an integration tests for incremental bricks sync once its complete
// with support for different profiles (go/jira/DECO-118)

// This test needs auth env vars to run.
// Please run using the deco env test or deco env shell
func TestAccFullSync(t *testing.T) {
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
	repoPath := fmt.Sprintf("/Repos/%s/%s", me.UserName, RandomName("empty-repo-sync-integration-"))

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

	// clone public empty remote repo
	tempDir := t.TempDir()
	cmd := exec.Command("git", "clone", repoUrl)
	cmd.Dir = tempDir
	err = cmd.Run()
	assert.NoError(t, err)

	// Create amsterdam.txt file
	projectDir := filepath.Join(tempDir, "empty-repo")
	f, err := os.Create(filepath.Join(projectDir, "amsterdam.txt"))
	assert.NoError(t, err)
	defer f.Close()

	// start bricks sync process
	cmd = exec.Command("go", "run", "main.go", "sync", "--remote-path", repoPath, "--persist-snapshot", "false")

	var cmdOut, cmdErr bytes.Buffer
	cmd.Stdout = &cmdOut
	cmd.Stderr = &cmdErr
	cmd.Dir = bricksRepo
	// bricks sync command will inherit the env vars from process
	os.Setenv("BRICKS_ROOT", projectDir)
	err = cmd.Start()
	assert.NoError(t, err)
	t.Cleanup(func() {
		// We wait three seconds to allow the bricks sync process flush its
		// stdout buffer
		time.Sleep(3 * time.Second)
		// terminate the bricks sync process
		cmd.Process.Kill()
		// Print the stdout and stderr logs from the bricks sync process
		fmt.Println("[INFO] bricks sync logs: ")
		if err != nil {
			fmt.Printf("error in bricks sync process: %s\n", err)
		}
		for _, line := range strings.Split(strings.TrimSuffix(cmdOut.String(), "\n"), "\n") {
			fmt.Println("[bricks sync stdout]", line)
		}
		for _, line := range strings.Split(strings.TrimSuffix(cmdErr.String(), "\n"), "\n") {
			fmt.Println("[bricks sync stderr]", line)
		}
	})

	// First upload assertion
	assert.Eventually(t, func() bool {
		repoContent, err := wsc.Workspace.List(ctx, workspace.ListRequest{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(repoContent.Objects) == 2
	}, 30*time.Second, 5*time.Second)
	repoContent, err := wsc.Workspace.List(ctx, workspace.ListRequest{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files1 []string
	for _, v := range repoContent.Objects {
		files1 = append(files1, filepath.Base(v.Path))
	}
	assert.Len(t, files1, 2)
	assert.Contains(t, files1, "amsterdam.txt")
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
	}, 30*time.Second, 5*time.Second)
	repoContent, err = wsc.Workspace.List(ctx, workspace.ListRequest{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files2 []string
	for _, v := range repoContent.Objects {
		files2 = append(files2, filepath.Base(v.Path))
	}
	assert.Len(t, files2, 4)
	assert.Contains(t, files2, "amsterdam.txt")
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
	}, 30*time.Second, 5*time.Second)
	repoContent, err = wsc.Workspace.List(ctx, workspace.ListRequest{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files3 []string
	for _, v := range repoContent.Objects {
		files3 = append(files3, filepath.Base(v.Path))
	}
	assert.Len(t, files3, 3)
	assert.Contains(t, files3, "amsterdam.txt")
	assert.Contains(t, files3, ".gitkeep")
	assert.Contains(t, files3, "world.txt")
}
