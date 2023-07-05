package acc

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/require"
)

type WorkspaceT struct {
	*testing.T

	W *databricks.WorkspaceClient

	ctx context.Context

	exec *compute.CommandExecutorV2
}

func WorkspaceTest(t *testing.T) (context.Context, *WorkspaceT) {
	loadDebugEnvIfRunFromIDE(t, "workspace")

	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	wt := &WorkspaceT{
		T: t,

		W: w,

		ctx: context.Background(),
	}

	return wt.ctx, wt
}

func (t *WorkspaceT) TestClusterID() string {
	clusterID := GetEnvOrSkipTest(t.T, "TEST_BRICKS_CLUSTER_ID")
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
