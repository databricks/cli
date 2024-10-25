package bundle

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccDashboards(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	warehouseID := acc.GetEnvOrSkipTest(t, "TEST_DEFAULT_WAREHOUSE_ID")
	uniqueID := uuid.New().String()
	root, err := initTestTemplate(t, ctx, "dashboards", map[string]any{
		"unique_id":    uniqueID,
		"warehouse_id": warehouseID,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = destroyBundle(t, ctx, root)
		require.NoError(t, err)
	})

	err = deployBundle(t, ctx, root)
	require.NoError(t, err)

	// Load bundle configuration by running the validate command.
	b := unmarshalConfig(t, mustValidateBundle(t, ctx, root))

	// Assert that the dashboard exists at the expected path and is, indeed, a dashboard.
	oi, err := wt.W.Workspace.GetStatusByPath(ctx, fmt.Sprintf("%s/test-dashboard-%s.lvdash.json", b.Config.Workspace.ResourcePath, uniqueID))
	require.NoError(t, err)
	assert.EqualValues(t, workspace.ObjectTypeDashboard, oi.ObjectType)

	// Load the dashboard by its ID and confirm its display name.
	dashboard, err := wt.W.Lakeview.GetByDashboardId(ctx, oi.ResourceId)
	require.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("test-dashboard-%s", uniqueID), dashboard.DisplayName)

	// Make an out of band modification to the dashboard and confirm that it is detected.
	_, err = wt.W.Lakeview.Update(ctx, dashboards.UpdateDashboardRequest{
		DashboardId:         oi.ResourceId,
		SerializedDashboard: dashboard.SerializedDashboard,
	})
	require.NoError(t, err)

	// Try to redeploy the bundle and confirm that the out of band modification is detected.
	stdout, _, err := deployBundleWithArgs(t, ctx, root)
	require.Error(t, err)
	assert.Contains(t, stdout, `Error: dashboard "file_reference" has been modified remotely`+"\n")

	// Redeploy the bundle with the --force flag and confirm that the out of band modification is ignored.
	_, stderr, err := deployBundleWithArgs(t, ctx, root, "--force")
	require.NoError(t, err)
	assert.Contains(t, stderr, `Deployment complete!`+"\n")
}
