package acc

import (
	"context"
	"sync"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/require"
)

type WorkspaceT struct {
	testutil.TestingT

	W *databricks.WorkspaceClient

	ctx context.Context

	exec *compute.CommandExecutorV2
}

// sharedWorkspaceClientState holds the workspace client shared by all tests in
// the process. A client per test repeats OIDC endpoint discovery for every
// test, and the SDK does not retry that request on timeout. When the workspace
// is briefly slow, every test starting at that moment fails with "cannot
// configure default credentials". A shared client authenticates once and
// refreshes its token in the background. The SDK does not cache a failed
// authentication, so the next call on the shared client tries again.
var sharedWorkspaceClientState struct {
	mu     sync.Mutex
	client *databricks.WorkspaceClient
}

func sharedWorkspaceClient(t testutil.TestingT) *databricks.WorkspaceClient {
	s := &sharedWorkspaceClientState
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		w, err := databricks.NewWorkspaceClient()
		require.NoError(t, err)
		s.client = w
	}
	return s.client
}

func WorkspaceTest(t testutil.TestingT) (context.Context, *WorkspaceT) {
	t.Helper()
	testutil.LoadDebugEnvIfRunFromIDE(t, "workspace")

	t.Logf("CLOUD_ENV=%s", testutil.GetEnvOrSkipTest(t, "CLOUD_ENV"))

	w := sharedWorkspaceClient(t)

	wt := &WorkspaceT{
		TestingT: t,

		W: w,

		ctx: t.Context(),
	}

	return wt.ctx, wt
}

func (t *WorkspaceT) TestClusterID() string {
	t.Helper()
	clusterID := testutil.GetEnvOrSkipTest(t, "TEST_BRICKS_CLUSTER_ID")
	err := t.W.Clusters.EnsureClusterIsRunning(t.ctx, clusterID)
	require.NoError(t, err, "Unexpected error from EnsureClusterIsRunning for clusterID=%s", clusterID)
	return clusterID
}

func (t *WorkspaceT) RunPython(code string) (string, error) {
	t.Helper()
	var err error

	// Create command executor only once per test.
	if t.exec == nil {
		t.exec, err = t.W.CommandExecution.Start(t.ctx, t.TestClusterID(), compute.LanguagePython)
		require.NoError(t, err, "Unexpected error from CommandExecution.Start(clusterID=%v)", t.TestClusterID())

		t.Cleanup(func() {
			err := t.exec.Destroy(context.WithoutCancel(t.ctx))
			require.NoError(t, err)
		})
	}

	results, err := t.exec.Execute(t.ctx, code)
	require.NoError(t, err, "Unexpected error from Execute(%v)", code)
	require.NotEqual(t, compute.ResultTypeError, results.ResultType, results.Cause)
	output, ok := results.Data.(string)
	require.True(t, ok, "unexpected type %T", results.Data)
	return output, nil
}
