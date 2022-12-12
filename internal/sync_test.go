package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/bricks/cmd/sync"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
)

// TODO: these tests are bloated, refactor these, and make write down tests for
// all edge cases with interop between files, directory and notebooks during syncing
// https://databricks.atlassian.net/browse/DECO-416

// This test needs auth env vars to run.
// Please run using the deco env test or deco env shell
func TestAccFullSync(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
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

	// Run `bricks sync` in the background.
	t.Setenv("BRICKS_ROOT", projectDir)
	c := NewCobraTestRunner(t, "sync", "--remote-path", repoPath, "--persist-snapshot=false")
	c.RunBackground()

	// First upload assertion
	c.Eventually(func() bool {
		objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(objects) == 3
	}, 30*time.Second, 5*time.Second)
	objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files1 []string
	for _, v := range objects {
		files1 = append(files1, filepath.Base(v.Path))
	}
	assert.Len(t, files1, 3)
	assert.Contains(t, files1, "amsterdam.txt")
	assert.Contains(t, files1, ".gitkeep")
	assert.Contains(t, files1, ".gitignore")

	// Create new files and assert
	os.Create(filepath.Join(projectDir, "hello.txt"))
	os.Create(filepath.Join(projectDir, "world.txt"))
	c.Eventually(func() bool {
		objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(objects) == 5
	}, 30*time.Second, 5*time.Second)
	objects, err = wsc.Workspace.ListAll(ctx, workspace.List{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files2 []string
	for _, v := range objects {
		files2 = append(files2, filepath.Base(v.Path))
	}
	assert.Len(t, files2, 5)
	assert.Contains(t, files2, "amsterdam.txt")
	assert.Contains(t, files2, ".gitkeep")
	assert.Contains(t, files2, "hello.txt")
	assert.Contains(t, files2, "world.txt")
	assert.Contains(t, files2, ".gitignore")

	// delete a file and assert
	os.Remove(filepath.Join(projectDir, "hello.txt"))
	c.Eventually(func() bool {
		objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(objects) == 4
	}, 30*time.Second, 5*time.Second)
	objects, err = wsc.Workspace.ListAll(ctx, workspace.List{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files3 []string
	for _, v := range objects {
		files3 = append(files3, filepath.Base(v.Path))
	}
	assert.Len(t, files3, 4)
	assert.Contains(t, files3, "amsterdam.txt")
	assert.Contains(t, files3, ".gitkeep")
	assert.Contains(t, files3, "world.txt")
	assert.Contains(t, files3, ".gitignore")
}

func assertSnapshotContents(t *testing.T, host, repoPath, projectDir string, listOfSyncedFiles []string) {
	snapshotPath := filepath.Join(projectDir, ".databricks/sync-snapshots", sync.GetFileName(host, repoPath))
	assert.FileExists(t, snapshotPath)

	var s *sync.Snapshot
	f, err := os.Open(snapshotPath)
	assert.NoError(t, err)
	defer f.Close()

	bytes, err := io.ReadAll(f)
	assert.NoError(t, err)
	err = json.Unmarshal(bytes, &s)
	assert.NoError(t, err)

	assert.Equal(t, s.Host, host)
	assert.Equal(t, s.RemotePath, repoPath)
	for _, filePath := range listOfSyncedFiles {
		_, ok := s.LastUpdatedTimes[filePath]
		assert.True(t, ok, fmt.Sprintf("%s not in snapshot file: %v", filePath, s.LastUpdatedTimes))
	}
	assert.Equal(t, len(listOfSyncedFiles), len(s.LastUpdatedTimes))
}

func TestAccIncrementalSync(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	wsc := databricks.Must(databricks.NewWorkspaceClient())
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

	// clone public empty remote repo
	tempDir := t.TempDir()
	cmd := exec.Command("git", "clone", repoUrl)
	cmd.Dir = tempDir
	err = cmd.Run()
	assert.NoError(t, err)

	projectDir := filepath.Join(tempDir, "empty-repo")

	// Add .databricks to .gitignore
	content := []byte("/.databricks/")
	f2, err := os.Create(filepath.Join(projectDir, ".gitignore"))
	assert.NoError(t, err)
	defer f2.Close()
	_, err = f2.Write(content)
	assert.NoError(t, err)

	// Run `bricks sync` in the background.
	t.Setenv("BRICKS_ROOT", projectDir)
	c := NewCobraTestRunner(t, "sync", "--remote-path", repoPath, "--persist-snapshot=true")
	c.RunBackground()

	// First upload assertion
	c.Eventually(func() bool {
		objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(objects) == 2
	}, 30*time.Second, 5*time.Second)
	objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files1 []string
	for _, v := range objects {
		files1 = append(files1, filepath.Base(v.Path))
	}
	assert.Len(t, files1, 2)
	assert.Contains(t, files1, ".gitignore")
	assert.Contains(t, files1, ".gitkeep")
	assertSnapshotContents(t, wsc.Config.Host, repoPath, projectDir, []string{".gitkeep", ".gitignore"})

	// Create amsterdam.txt file
	f, err := os.Create(filepath.Join(projectDir, "amsterdam.txt"))
	assert.NoError(t, err)
	defer f.Close()

	// new file upload assertion
	c.Eventually(func() bool {
		objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(objects) == 3
	}, 30*time.Second, 5*time.Second)
	objects, err = wsc.Workspace.ListAll(ctx, workspace.List{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files2 []string
	for _, v := range objects {
		files2 = append(files2, filepath.Base(v.Path))
	}
	assert.Len(t, files2, 3)
	assert.Contains(t, files2, "amsterdam.txt")
	assert.Contains(t, files2, ".gitkeep")
	assert.Contains(t, files2, ".gitignore")
	assertSnapshotContents(t, wsc.Config.Host, repoPath, projectDir, []string{"amsterdam.txt", ".gitkeep", ".gitignore"})

	// delete a file and assert
	os.Remove(filepath.Join(projectDir, ".gitkeep"))
	c.Eventually(func() bool {
		objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(objects) == 2
	}, 30*time.Second, 5*time.Second)
	objects, err = wsc.Workspace.ListAll(ctx, workspace.List{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files3 []string
	for _, v := range objects {
		files3 = append(files3, filepath.Base(v.Path))
	}
	assert.Len(t, files3, 2)
	assert.Contains(t, files3, "amsterdam.txt")
	assert.Contains(t, files3, ".gitignore")
	assertSnapshotContents(t, wsc.Config.Host, repoPath, projectDir, []string{"amsterdam.txt", ".gitignore"})

	// new file in dir upload assertion
	fooPath := filepath.Join(projectDir, "bar/foo.txt")
	err = os.MkdirAll(filepath.Dir(fooPath), os.ModePerm)
	assert.NoError(t, err)
	f, err = os.Create(fooPath)
	assert.NoError(t, err)
	defer f.Close()
	assert.Eventually(t, func() bool {
		objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(objects) == 3
	}, 30*time.Second, 5*time.Second)
	objects, err = wsc.Workspace.ListAll(ctx, workspace.List{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files4 []string
	for _, v := range objects {
		files4 = append(files4, filepath.Base(v.Path))
	}
	assert.Len(t, files4, 3)
	assert.Contains(t, files4, "amsterdam.txt")
	assert.Contains(t, files4, ".gitignore")
	assert.Contains(t, files4, "bar")
	assertSnapshotContents(t, wsc.Config.Host, repoPath, projectDir, []string{"amsterdam.txt", "bar/foo.txt", ".gitignore"})

	// delete dir
	err = os.RemoveAll(filepath.Dir(fooPath))
	assert.NoError(t, err)
	assert.Eventually(t, func() bool {
		objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(objects) == 3
	}, 30*time.Second, 5*time.Second)
	objects, err = wsc.Workspace.ListAll(ctx, workspace.List{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files5 []string
	for _, v := range objects {
		files5 = append(files5, filepath.Base(v.Path))
		if strings.Contains(v.Path, "bar") {
			assert.Equal(t, workspace.ObjectType("DIRECTORY"), v.ObjectType)
		}
	}
	assert.Len(t, files5, 3)
	assert.Contains(t, files5, "bar")
	assert.Contains(t, files5, "amsterdam.txt")
	assert.Contains(t, files5, ".gitignore")
	// workspace still contains `bar` directory but it has been deleted from snapshot
	assertSnapshotContents(t, wsc.Config.Host, repoPath, projectDir, []string{"amsterdam.txt", ".gitignore"})

	// file called bar should overwrite the directory
	err = os.WriteFile(filepath.Join(projectDir, "bar"), []byte("Kal ho na ho is a cool movie"), os.ModePerm)
	assert.NoError(t, err)
	assert.Eventually(t, func() bool {
		objects, err := wsc.Workspace.ListAll(ctx, workspace.List{
			Path: repoPath,
		})
		assert.NoError(t, err)
		return len(objects) == 3
	}, 30*time.Second, 5*time.Second)
	objects, err = wsc.Workspace.ListAll(ctx, workspace.List{
		Path: repoPath,
	})
	assert.NoError(t, err)
	var files6 []string
	for _, v := range objects {
		files6 = append(files6, filepath.Base(v.Path))
		if strings.Contains(v.Path, "bar") {
			assert.Equal(t, workspace.ObjectType("FILE"), v.ObjectType)
		}
	}
	assert.Len(t, files6, 3)
	assert.Contains(t, files6, "amsterdam.txt")
	assert.Contains(t, files6, ".gitignore")
	// workspace still contains `bar` directory but it has been deleted from snapshot
	assert.Contains(t, files6, "bar")
	assertSnapshotContents(t, wsc.Config.Host, repoPath, projectDir, []string{"amsterdam.txt", "bar", ".gitignore"})
}
