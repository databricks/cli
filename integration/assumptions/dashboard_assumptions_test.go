package assumptions_test

import (
	"encoding/base64"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Verify that importing a dashboard through the Workspace API retains the identity of the underying resource,
// as well as properties exclusively accessible through the dashboards API.
func TestDashboardAssumptions_WorkspaceImport(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	t.Parallel()

	dashboardName := "New Dashboard"
	dashboardPayload := []byte(`{"pages":[{"name":"2506f97a","displayName":"New Page"}]}`)
	warehouseId := testutil.GetEnvOrSkipTest(t, "TEST_DEFAULT_WAREHOUSE_ID")

	dir := acc.TemporaryWorkspaceDir(wt, "dashboard-assumptions-")

	dashboard, err := wt.W.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{
		Dashboard: dashboards.Dashboard{
			DisplayName:         dashboardName,
			ParentPath:          dir,
			SerializedDashboard: string(dashboardPayload),
			WarehouseId:         warehouseId,
		},
	})
	require.NoError(t, err)
	t.Logf("Dashboard ID (per Lakeview API): %s", dashboard.DashboardId)

	// Overwrite the dashboard via the workspace API.
	{
		err := wt.W.Workspace.Import(ctx, workspace.Import{
			Format:    workspace.ImportFormatAuto,
			Path:      dashboard.Path,
			Content:   base64.StdEncoding.EncodeToString(dashboardPayload),
			Overwrite: true,
		})
		require.NoError(t, err)
	}

	// Cross-check consistency with the workspace object.
	{
		obj, err := wt.W.Workspace.GetStatusByPath(ctx, dashboard.Path) //nolint:staticcheck // Deprecated in SDK v0.127.0. Migration to WorkspaceHierarchyService tracked separately.
		require.NoError(t, err)

		// Confirm that the resource ID included in the response is equal to the dashboard ID.
		require.Equal(t, dashboard.DashboardId, obj.ResourceId)
		t.Logf("Dashboard ID (per workspace object status): %s", obj.ResourceId)
	}

	// Try to overwrite the dashboard via the Lakeview API (and expect failure).
	{
		_, err := wt.W.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{
			Dashboard: dashboards.Dashboard{
				DisplayName:         dashboardName,
				ParentPath:          dir,
				SerializedDashboard: string(dashboardPayload),
			},
		})
		// Lakeview returns the generic gRPC error_code ALREADY_EXISTS, not
		// Databricks' RESOURCE_ALREADY_EXISTS, so the SDK unwraps to
		// ErrAlreadyExists rather than ErrResourceAlreadyExists. Assert the
		// common 409 parent to stay resilient to either code.
		require.ErrorIs(t, err, apierr.ErrResourceConflict)
	}

	// Retrieve the dashboard object and confirm that only select fields were updated by the import.
	{
		previousDashboard := dashboard
		currentDashboard, err := wt.W.Lakeview.Get(ctx, dashboards.GetDashboardRequest{
			DashboardId: dashboard.DashboardId,
		})
		require.NoError(t, err)

		// Convert the dashboard object to a [dyn.Value] to make comparison easier.
		previous, err := convert.FromTyped(previousDashboard, dyn.NilValue)
		require.NoError(t, err)
		current, err := convert.FromTyped(currentDashboard, dyn.NilValue)
		require.NoError(t, err)

		// Collect updated and deleted paths.
		var updatedFieldPaths []string
		var deletedFieldPaths []string
		_, err = merge.Override(previous, current, merge.OverrideVisitor{
			VisitDelete: func(basePath dyn.Path, left dyn.Value) error {
				deletedFieldPaths = append(deletedFieldPaths, basePath.String())
				return nil
			},
			VisitInsert: func(basePath dyn.Path, right dyn.Value) (dyn.Value, error) {
				assert.Fail(t, "unexpected insert operation")
				return right, nil
			},
			VisitUpdate: func(basePath dyn.Path, left, right dyn.Value) (dyn.Value, error) {
				updatedFieldPaths = append(updatedFieldPaths, basePath.String())
				return right, nil
			},
		})
		require.NoError(t, err)

		// etag and update_time always change after workspace import. serialized_dashboard
		// may also appear depending on the Lakeview server version (observed on AWS staging
		// but not on GCP production).
		assert.Contains(t, updatedFieldPaths, "etag")
		assert.Contains(t, updatedFieldPaths, "update_time")
		assert.Subset(t, []string{"etag", "update_time", "serialized_dashboard"}, updatedFieldPaths)

		// warehouse_id may be cleared depending on the Lakeview server version (observed on
		// AWS staging but not on GCP production).
		assert.Subset(t, []string{"warehouse_id"}, deletedFieldPaths)
	}
}
