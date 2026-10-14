package acc

import (
	"context"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"
)

type WorkspaceT struct {
	testutil.TestingT

	W *sdkclients.Clients

	ctx context.Context
}

func WorkspaceTest(t testutil.TestingT) (context.Context, *WorkspaceT) {
	t.Helper()
	testutil.LoadDebugEnvIfRunFromIDE(t, "workspace")

	t.Logf("CLOUD_ENV=%s", testutil.GetEnvOrSkipTest(t, "CLOUD_ENV"))

	w, err := sdkclients.New(t.Context(), &config.Config{})
	require.NoError(t, err)

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
	err := t.W.EnsureClusterIsRunning(t.ctx, clusterID)
	require.NoError(t, err, "Unexpected error from EnsureClusterIsRunning for clusterID=%s", clusterID)
	return clusterID
}
