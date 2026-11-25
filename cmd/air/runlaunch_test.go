package aircmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCliLaunchDir(t *testing.T) {
	dir := cliLaunchDir("/Workspace/Users/me@example.com", "my-exp", "")
	assert.True(t, strings.HasPrefix(dir, "/Workspace/Users/me@example.com/.air/cli_launch/my-exp/my-exp_"), dir)
	// run name overrides the leaf; the unique suffix keeps successive dirs distinct.
	withRun := cliLaunchDir("/base", "exp", "run1")
	assert.True(t, strings.HasPrefix(withRun, "/base/.air/cli_launch/exp/run1_"), withRun)
	assert.NotEqual(t, dir, cliLaunchDir("/Workspace/Users/me@example.com", "my-exp", ""))
}

func newFakeWorkspaceClient(t *testing.T) *databricks.WorkspaceClient {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	return w
}

func TestUserWorkspaceDir(t *testing.T) {
	w := newFakeWorkspaceClient(t)
	t.Setenv(userWorkspaceDirEnv, "")
	dir, err := userWorkspaceDir(t.Context(), w)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(dir, "/Workspace/Users/"), dir)

	// The env override wins without an API call.
	t.Setenv(userWorkspaceDirEnv, "/Workspace/custom")
	dir, err = userWorkspaceDir(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/custom", dir)
}

func TestUserWorkspaceDirRequestsOnlyUserName(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	var attributes []string
	server.Handle("GET", "/api/2.0/preview/scim/v2/Me", func(req testserver.Request) any {
		attributes = append(attributes, req.URL.Query().Get("attributes"))
		return iam.User{UserName: "me@example.com"}
	})
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
	require.NoError(t, err)
	t.Setenv(userWorkspaceDirEnv, "")

	dir, err := userWorkspaceDir(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/Users/me@example.com", dir)
	assert.Equal(t, []string{"userName"}, attributes)
}

func TestUserWorkspaceDirProfilePrecedence(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(configPath, []byte(`
[__settings__]
default_profile = second

[first]
host = https://first.test
databricks_internal_user_workspace_dir = /Workspace/first

[second]
host = https://second.test
databricks_internal_user_workspace_dir = /Workspace/second
`), 0o600))

	t.Run("explicit profile", func(t *testing.T) {
		w := newFakeWorkspaceClient(t)
		w.Config.ConfigFile = configPath
		w.Config.Profile = "first"
		t.Setenv(userWorkspaceDirEnv, "")
		dir, err := userWorkspaceDir(t.Context(), w)
		require.NoError(t, err)
		assert.Equal(t, "/Workspace/first", dir)
	})

	t.Run("default profile", func(t *testing.T) {
		w := newFakeWorkspaceClient(t)
		w.Config.ConfigFile = configPath
		w.Config.Profile = ""
		t.Setenv(userWorkspaceDirEnv, "")
		dir, err := userWorkspaceDir(t.Context(), w)
		require.NoError(t, err)
		assert.Equal(t, "/Workspace/second", dir)
	})

	t.Run("environment wins", func(t *testing.T) {
		w := newFakeWorkspaceClient(t)
		w.Config.ConfigFile = configPath
		w.Config.Profile = "first"
		t.Setenv(userWorkspaceDirEnv, "/Workspace/env")
		dir, err := userWorkspaceDir(t.Context(), w)
		require.NoError(t, err)
		assert.Equal(t, "/Workspace/env", dir)
	})
}

// checkExperimentDirectory reads dir and creates it if missing, the way
// submitWorkload does across its two steps.
func checkExperimentDirectory(t *testing.T, w *databricks.WorkspaceClient, dir string) error {
	t.Helper()
	return ensureExperimentDirectory(t.Context(), w, dir, awaitExperimentDirRead(t, startExperimentDirectoryRead(t.Context(), w, dir)))
}

// awaitExperimentDirRead waits for the background read, or fails the test after a
// few seconds so a regression fails instead of hanging.
func awaitExperimentDirRead(t *testing.T, read <-chan experimentDirStatus) experimentDirStatus {
	t.Helper()
	select {
	case status := <-read:
		return status
	case <-time.After(5 * time.Second):
		t.Fatal("experiment directory read did not finish")
		return experimentDirStatus{}
	}
}

func TestEnsureExperimentDirectory(t *testing.T) {
	ctx := t.Context()
	w := newFakeWorkspaceClient(t)

	// Empty means default (always exists) — no API call, no error.
	require.NoError(t, checkExperimentDirectory(t, w, ""))

	// A missing path is created.
	require.NoError(t, checkExperimentDirectory(t, w, "/Workspace/Users/me/exp"))

	// An existing directory is accepted as-is.
	require.NoError(t, w.Workspace.MkdirsByPath(ctx, "/Workspace/Users/me/existing"))
	require.NoError(t, checkExperimentDirectory(t, w, "/Workspace/Users/me/existing"))

	// A path that exists but is a file is rejected.
	fc, err := filer.NewWorkspaceFilesClient(w, "/Workspace/Users/me")
	require.NoError(t, err)
	require.NoError(t, fc.Write(ctx, "afile", strings.NewReader("x")))
	err = checkExperimentDirectory(t, w, "/Workspace/Users/me/afile")
	require.ErrorContains(t, err, "is not a directory")
}

// experimentDirServer answers get-status 404 for a missing directory, 500 for any
// path in failPaths, and counts mkdirs calls.
func experimentDirServer(t *testing.T, failPaths map[string]bool, mkdirs *int) *databricks.WorkspaceClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/workspace/mkdirs":
			*mkdirs++
			_, _ = w.Write([]byte(`{}`))
		case "/api/2.0/workspace/get-status":
			if failPaths[r.URL.Query().Get("path")] {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error_code":"INTERNAL_ERROR","message":"boom"}`))
				return
			}
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"not found"}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return newTestWorkspaceClient(t, srv.URL)
}

func TestExperimentDirectoryReadCreatesNothing(t *testing.T) {
	const dir = "/Workspace/Users/me/exp"
	var mkdirs int
	w := experimentDirServer(t, nil, &mkdirs)

	// The read may run before Me; only ensureExperimentDirectory writes.
	status := awaitExperimentDirRead(t, startExperimentDirectoryRead(t.Context(), w, dir))
	assert.Equal(t, 0, mkdirs, "the finished read wrote nothing")
	require.NoError(t, ensureExperimentDirectory(t.Context(), w, dir, status))
	assert.Equal(t, 1, mkdirs, "the missing directory is created after the read")
}

func TestExperimentDirectoryReadErrorSurfaces(t *testing.T) {
	const dir = "/Workspace/Users/me/exp"
	var mkdirs int
	w := experimentDirServer(t, map[string]bool{dir: true}, &mkdirs)
	err := checkExperimentDirectory(t, w, dir)
	require.ErrorContains(t, err, "failed to check experiment_directory")
	assert.Equal(t, 0, mkdirs, "a failed read creates nothing")
}

// launchFilesServer answers get-status with the size of each path in files and
// 404 for any other path.
func launchFilesServer(t *testing.T, files map[string]int) *databricks.WorkspaceClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("path")
		size, ok := files[p]
		if r.URL.Path != "/api/2.0/workspace/get-status" || !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"not found"}`))
			return
		}
		_, _ = fmt.Fprintf(w, `{"path":%q,"object_type":"FILE","size":%d}`, p, size)
	}))
	t.Cleanup(srv.Close)
	return newTestWorkspaceClient(t, srv.URL)
}

func TestLaunchFilesStaged(t *testing.T) {
	const dir = "/Workspace/Users/me@example.com/.air/cli_launch/exp/run_abc"
	items := []uploadItem{{trainingConfigName, []byte("cfg")}, {commandScriptName, []byte("echo hi")}}
	// staged probes a workspace that holds items as uploaded, after edit.
	staged := func(edit func(files map[string]int)) bool {
		files := map[string]int{}
		for _, item := range items {
			files[path.Join(dir, item.name)] = len(item.data)
		}
		edit(files)
		return launchFilesStaged(t.Context(), launchFilesServer(t, files), dir, items)
	}
	assert.True(t, staged(func(map[string]int) {}))
	// A partly written dir, or a file whose size changed in the workspace, is a miss.
	for _, item := range items {
		p := path.Join(dir, item.name)
		assert.False(t, staged(func(files map[string]int) { delete(files, p) }), item.name+" missing")
		assert.False(t, staged(func(files map[string]int) { files[p]++ }), item.name+" resized")
	}
}

func TestLaunchContentHash(t *testing.T) {
	files := []uploadItem{{commandScriptName, []byte("echo hi")}, {trainingConfigName, []byte("cfg")}}
	assert.Equal(t, launchContentHash(files), launchContentHash([]uploadItem{{commandScriptName, []byte("echo hi")}, {trainingConfigName, []byte("cfg")}}))
	assert.NotEqual(t, launchContentHash(files), launchContentHash([]uploadItem{{commandScriptName, []byte("echo bye")}, {trainingConfigName, []byte("cfg")}}))
	assert.NotEqual(t, launchContentHash([]uploadItem{{"a", []byte("bc")}}), launchContentHash([]uploadItem{{"ab", []byte("c")}}))
}
