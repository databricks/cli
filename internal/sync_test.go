package internal

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/databricks/bricks/cmd/root"
	"github.com/databricks/bricks/cmd/sync"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/repos"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type syncTestHarness struct {
	*testing.T

	errch <-chan error
}

// Run CLI in-process and asynchronously by invoking [RootCmd] directly.
// Upon termination, either a nil or an error is sent to [errch] and the channel is closed.
func (s *syncTestHarness) run(args ...string) {
	ctx, cancel := context.WithCancel(context.Background())

	cmd := root.RootCmd
	cmd.SetArgs(args)

	// Consolidate stdout and stderr in single buffer.
	var outerr bytes.Buffer
	cmd.SetOut(&outerr)
	cmd.SetErr(&outerr)

	errch := make(chan error)

	// Run sync command in background.
	go func() {
		err := cmd.ExecuteContext(ctx)
		if err != nil {
			s.Logf("Error running command: %s", err)
		}

		// Log everything printed by the command.
		scanner := bufio.NewScanner(&outerr)
		for scanner.Scan() {
			s.Logf("[bricks output]: %s", scanner.Text())
		}

		// Make caller aware of error.
		errch <- err
		close(errch)
	}()

	// Terminate command (if necessary).
	s.Cleanup(func() {
		// Signal termination of command.
		cancel()
		// Wait for goroutine to finish.
		<-errch
	})

	s.errch = errch
}

// Like [s.eventually] but errors if the underlying command has failed.
func (s *syncTestHarness) eventually(condition func() bool, waitFor time.Duration, tick time.Duration, msgAndArgs ...interface{}) bool {
	ch := make(chan bool, 1)

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	ticker := time.NewTicker(tick)
	defer ticker.Stop()

	for tick := ticker.C; ; {
		select {
		case err := <-s.errch:
			require.Fail(s, "Command failed", err)
		case <-timer.C:
			require.Fail(s, "Condition never satisfied", msgAndArgs...)
		case <-tick:
			tick = nil
			go func() { ch <- condition() }()
		case v := <-ch:
			if v {
				return true
			}
			tick = ticker.C
		}
	}
}

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
	s := &syncTestHarness{T: t}
	s.run("sync", "--remote-path", repoPath, "--persist-snapshot=false")

	// First upload assertion
	s.eventually(func() bool {
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
	s.eventually(func() bool {
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
	s.eventually(func() bool {
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
	s := &syncTestHarness{T: t}
	s.run("sync", "--remote-path", repoPath, "--persist-snapshot=true")

	// First upload assertion
	s.eventually(func() bool {
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
	s.eventually(func() bool {
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
	s.eventually(func() bool {
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
}
