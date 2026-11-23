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

func TestEnsureExperimentDirectory(t *testing.T) {
	ctx := t.Context()
	w := newFakeWorkspaceClient(t)

	// Empty means default (always exists) — no API call, no error.
	require.NoError(t, ensureExperimentDirectory(ctx, w, ""))

	// A missing path is created.
	require.NoError(t, ensureExperimentDirectory(ctx, w, "/Workspace/Users/me/exp"))

	// An existing directory is accepted as-is.
	require.NoError(t, w.Workspace.MkdirsByPath(ctx, "/Workspace/Users/me/existing"))
	require.NoError(t, ensureExperimentDirectory(ctx, w, "/Workspace/Users/me/existing"))

	// A path that exists but is a file is rejected.
	fc, err := filer.NewWorkspaceFilesClient(w, "/Workspace/Users/me")
	require.NoError(t, err)
	require.NoError(t, fc.Write(ctx, "afile", strings.NewReader("x")))
	err = ensureExperimentDirectory(ctx, w, "/Workspace/Users/me/afile")
	require.ErrorContains(t, err, "is not a directory")
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
