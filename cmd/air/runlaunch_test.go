package aircmd

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cache"
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

// launchFilesServer answers get-status 200 for every path except missing.
func launchFilesServer(t *testing.T, missing *string) *databricks.WorkspaceClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Query().Get("path")
		if r.URL.Path != "/api/2.0/workspace/get-status" || p == *missing {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"not found"}`))
			return
		}
		_, _ = w.Write([]byte(`{"path":"` + p + `","object_type":"FILE"}`))
	}))
	t.Cleanup(srv.Close)
	return newTestWorkspaceClient(t, srv.URL)
}

func TestLaunchFilesStaged(t *testing.T) {
	const dir = "/Workspace/Users/me@example.com/.air/cli_launch/exp/run_abc"
	items := []uploadItem{{trainingConfigName, []byte("cfg")}, {commandScriptName, []byte("echo hi")}}
	missing := ""
	w := launchFilesServer(t, &missing)
	assert.True(t, launchFilesStaged(t.Context(), w, dir, items))
	// A partly written dir is a miss, whichever file is absent.
	for _, item := range items {
		missing = path.Join(dir, item.name)
		assert.False(t, launchFilesStaged(t.Context(), w, dir, items), item.name)
	}
}

func TestLaunchContentHash(t *testing.T) {
	files := []uploadItem{{commandScriptName, []byte("echo hi")}, {trainingConfigName, []byte("cfg")}}
	assert.Equal(t, launchContentHash(files), launchContentHash([]uploadItem{{commandScriptName, []byte("echo hi")}, {trainingConfigName, []byte("cfg")}}))
	assert.NotEqual(t, launchContentHash(files), launchContentHash([]uploadItem{{commandScriptName, []byte("echo bye")}, {trainingConfigName, []byte("cfg")}}))
	assert.NotEqual(t, launchContentHash([]uploadItem{{"a", []byte("bc")}}), launchContentHash([]uploadItem{{"ab", []byte("c")}}))
}

func TestLaunchIndex(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("DATABRICKS_CACHE_ENABLED", "true")
	ctx := t.Context()
	key := launchIndexKey{Host: "https://h.example.com", Base: "/Workspace/Users/me", Content: "c1"}
	const dir = "/Workspace/Users/me/.air/cli_launch/exp/run_1"

	_, ok := cache.Get[string](ctx, newLaunchIndex(ctx), key)
	assert.False(t, ok, "first launch has no entry")

	cache.Put(ctx, newLaunchIndex(ctx), key, dir)
	got, ok := cache.Get[string](ctx, newLaunchIndex(ctx), key)
	require.True(t, ok)
	assert.Equal(t, dir, got)

	// Another workspace never sees this entry.
	other := key
	other.Host = "https://other.example.com"
	_, ok = cache.Get[string](ctx, newLaunchIndex(ctx), other)
	assert.False(t, ok)

	// A corrupt entry reads as a miss, so the launch takes the first-launch path.
	require.NoError(t, filepath.WalkDir(os.Getenv("XDG_CACHE_HOME"), func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			return os.WriteFile(p, []byte("{not json"), 0o600)
		}
		return err
	}))
	_, ok = cache.Get[string](ctx, newLaunchIndex(ctx), key)
	assert.False(t, ok)
}
