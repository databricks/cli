package acc

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

type WorkspaceT struct {
	testutil.TestingT

	W *databricks.WorkspaceClient

	ctx context.Context

	exec *compute.CommandExecutorV2
}

func WorkspaceTest(t testutil.TestingT) (context.Context, *WorkspaceT) {
	loadDebugEnvIfRunFromIDE(t, "workspace")

	t.Log(testutil.GetEnvOrSkipTest(t, "CLOUD_ENV"))

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	wt := &WorkspaceT{
		TestingT: t,

		W: w,

		ctx: context.Background(),
	}

	return wt.ctx, wt
}

// Run the workspace test only on UC workspaces.
func UcWorkspaceTest(t testutil.TestingT) (context.Context, *WorkspaceT) {
	loadDebugEnvIfRunFromIDE(t, "workspace")

	t.Log(testutil.GetEnvOrSkipTest(t, "CLOUD_ENV"))

	if os.Getenv("TEST_METASTORE_ID") == "" {
		t.Skipf("Skipping on non-UC workspaces")
	}
	if os.Getenv("DATABRICKS_ACCOUNT_ID") != "" {
		t.Skipf("Skipping on accounts")
	}

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	wt := &WorkspaceT{
		TestingT: t,

		W: w,

		ctx: context.Background(),
	}

	return wt.ctx, wt
}

func (t *WorkspaceT) TestClusterID() string {
	clusterID := testutil.GetEnvOrSkipTest(t, "TEST_BRICKS_CLUSTER_ID")
	err := t.W.Clusters.EnsureClusterIsRunning(t.ctx, clusterID)
	require.NoError(t, err)
	return clusterID
}

func (t *WorkspaceT) RunPython(code string) (string, error) {
	var err error

	// Create command executor only once per test.
	if t.exec == nil {
		t.exec, err = t.W.CommandExecution.Start(t.ctx, t.TestClusterID(), compute.LanguagePython)
		require.NoError(t, err)

		t.Cleanup(func() {
			err := t.exec.Destroy(t.ctx)
			require.NoError(t, err)
		})
	}

	results, err := t.exec.Execute(t.ctx, code)
	require.NoError(t, err)
	require.NotEqual(t, compute.ResultTypeError, results.ResultType, results.Cause)
	output, ok := results.Data.(string)
	require.True(t, ok, "unexpected type %T", results.Data)
	return output, nil
}

func (t *WorkspaceT) TemporaryWorkspaceDir(name ...string) string {
	ctx := context.Background()
	me, err := t.W.CurrentUser.Me(ctx)
	require.NoError(t, err)

	basePath := fmt.Sprintf("/Users/%s/%s", me.UserName, testutil.RandomName(name...))

	t.Logf("Creating %s", basePath)
	err = t.W.Workspace.MkdirsByPath(ctx, basePath)
	require.NoError(t, err)

	// Remove test directory on test completion.
	t.Cleanup(func() {
		t.Logf("Removing %s", basePath)
		err := t.W.Workspace.Delete(ctx, workspace.Delete{
			Path:      basePath,
			Recursive: true,
		})
		if err == nil || apierr.IsMissing(err) {
			return
		}
		t.Logf("Unable to remove temporary workspace directory %s: %#v", basePath, err)
	})

	return basePath
}
