package aircmd

import (
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
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
	dir, err := userWorkspaceDir(t.Context(), w)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(dir, "/Workspace/Users/"), dir)

	// The env override wins without an API call.
	t.Setenv(userWorkspaceDirEnv, "/Workspace/custom")
	dir, err = userWorkspaceDir(t.Context(), w)
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/custom", dir)
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
