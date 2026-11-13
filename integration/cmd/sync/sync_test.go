package sync_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/sync"
	"github.com/databricks/cli/libs/testfile"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var repoFiles = []string{}

type syncTest struct {
	t          *testing.T
	c          *testcli.Runner
	w          *databricks.WorkspaceClient
	localRoot  string
	remoteRoot string
}

func setupSyncTest(t *testing.T, args ...string) (context.Context, *syncTest) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	localRoot := t.TempDir()
	remoteRoot := acc.TemporaryWorkspaceDir(wt, "sync-")
	// Prepend common arguments.
	args = append([]string{
		"sync",
		localRoot,
		remoteRoot,
		"--output",
		"json",
	}, args...)

	c := testcli.NewRunner(t, ctx, args...)
	c.RunBackground()

	return ctx, &syncTest{
		t:          t,
		c:          c,
		w:          w,
		localRoot:  localRoot,
		remoteRoot: remoteRoot,
	}
}

func (a *syncTest) waitForCompletionMarker() {
	ctx, cancel := context.WithTimeout(a.t.Context(), 60*time.Second)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			a.t.Fatal("timed out waiting for sync to complete")
		case line := <-a.c.StdoutLines:
			var event sync.EventBase
			err := json.Unmarshal([]byte(line), &event)
			require.NoError(a.t, err)
			if event.Type == sync.EventTypeComplete {
				return
			}
		}
	}
}

func (a *syncTest) remoteDirContent(ctx context.Context, relativeDir string, expectedFiles []string) {
	remoteDir := path.Join(a.remoteRoot, relativeDir)
	a.c.Eventually(func() bool {
		objects, err := a.w.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{
			Path: remoteDir,
		})
		require.NoError(a.t, err)
		return len(objects) == len(expectedFiles)
	}, 30*time.Second, 5*time.Second)
	objects, err := a.w.Workspace.ListAll(ctx, workspace.ListWorkspaceRequest{
		Path: remoteDir,
	})
	require.NoError(a.t, err)

	var actualFiles []string
	for _, v := range objects {
		actualFiles = append(actualFiles, v.Path)
	}

	assert.Len(a.t, actualFiles, len(expectedFiles))
	for _, v := range expectedFiles {
		assert.Contains(a.t, actualFiles, path.Join(a.remoteRoot, relativeDir, v))
	}
}

func (a *syncTest) remoteFileContent(ctx context.Context, relativePath, expectedContent string) {
	filePath := path.Join(a.remoteRoot, relativePath)

	// Remove leading "/" so we can use it in the URL.
	urlPath := "/api/2.0/workspace-files/" + strings.TrimLeft(filePath, "/")

	apiClient, err := client.New(a.w.Config)
	require.NoError(a.t, err)

	var res []byte
	a.c.Eventually(func() bool {
		err = apiClient.Do(ctx, http.MethodGet, urlPath, nil, nil, nil, &res)
		require.NoError(a.t, err)
		actualContent := string(res)
		return actualContent == expectedContent
	}, 30*time.Second, 5*time.Second)
}

func (a *syncTest) snapshotContains(files []string) {
	snapshotPath := filepath.Join(a.localRoot, ".databricks/sync-snapshots", sync.GetFileName(a.w.Config.Host, a.remoteRoot))
	assert.FileExists(a.t, snapshotPath)

	var s *sync.Snapshot
	f, err := os.Open(snapshotPath)
	assert.NoError(a.t, err)
	defer f.Close()

	bytes, err := io.ReadAll(f)
	assert.NoError(a.t, err)
	err = json.Unmarshal(bytes, &s)
	assert.NoError(a.t, err)

	assert.Equal(a.t, s.Host, a.w.Config.Host)
	assert.Equal(a.t, s.RemotePath, a.remoteRoot)
	for _, filePath := range files {
		_, ok := s.LastModifiedTimes[filePath]
		assert.True(a.t, ok, "%s not in snapshot file: %v", filePath, s.LastModifiedTimes)
	}
	assert.Len(a.t, s.LastModifiedTimes, len(files), "files=%s s.LastModifiedTimes=%s", files, s.LastModifiedTimes)
}

// Other sync scenarios are acceptance tests under acceptance/cmd/sync-incremental, each step a
// separate CLI run. This test stays here to cover --watch, which runs as one long-lived process.
func TestSyncIncrementalFileSync(t *testing.T) {
	ctx, assertSync := setupSyncTest(t, "--watch")

	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)

	// New file
	localFilePath := filepath.Join(assertSync.localRoot, "foo.txt")
	f := testfile.CreateFile(t, localFilePath)
	defer f.Close(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", append(repoFiles, "foo.txt"))
	assertSync.remoteFileContent(ctx, "foo.txt", "")
	assertSync.snapshotContains(append(repoFiles, "foo.txt"))

	// Write to file
	f.Overwrite(t, `{"statement": "Mi Gente"}`)
	assertSync.waitForCompletionMarker()
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Mi Gente"}`)

	// Write again
	f.Overwrite(t, `{"statement": "Young Dumb & Broke"}`)
	assertSync.waitForCompletionMarker()
	assertSync.remoteFileContent(ctx, "foo.txt", `{"statement": "Young Dumb & Broke"}`)

	// delete
	f.Remove(t)
	assertSync.waitForCompletionMarker()
	assertSync.remoteDirContent(ctx, "", repoFiles)
	assertSync.snapshotContains(repoFiles)
}
