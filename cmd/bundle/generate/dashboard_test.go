package generate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/databricks/databricks-sdk-go/service/workspace"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestDashboard_ErrorOnLegacyDashboard(t *testing.T) {
	// Response to a GetStatus request on a path pointing to a legacy dashboard.
	//
	// < HTTP/2.0 400 Bad Request
	// < {
	// <   "error_code": "BAD_REQUEST",
	// <   "message": "dbsqlDashboard is not user-facing."
	// < }

	d := dashboard{
		existingPath: "/path/to/legacy dashboard",
	}

	m := mocks.NewMockWorkspaceClient(t)
	w := m.GetMockWorkspaceAPI()
	w.On("GetStatusByPath", mock.Anything, "/path/to/legacy dashboard").Return(nil, &apierr.APIError{
		StatusCode: 400,
		ErrorCode:  "BAD_REQUEST",
		Message:    "dbsqlDashboard is not user-facing.",
	})

	ctx := context.Background()
	b := &bundle.Bundle{}
	b.SetWorkpaceClient(m.WorkspaceClient)

	_, diags := d.resolveID(ctx, b)
	require.Len(t, diags, 1)
	assert.Equal(t, "dashboard \"legacy dashboard\" is a legacy dashboard", diags[0].Summary)
}

func TestDashboard_ExistingID_Nominal(t *testing.T) {
	root := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: root,
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	dashboardsAPI := m.GetMockLakeviewAPI()
	dashboardsAPI.EXPECT().GetByDashboardId(mock.Anything, "f00dcafe").Return(&dashboards.Dashboard{
		DashboardId:         "f00dcafe",
		DisplayName:         "This is a test dashboard",
		SerializedDashboard: `{"pages":[{"displayName":"New Page","layout":[],"name":"12345678"}]}`,
		WarehouseId:         "w4r3h0us3",
	}, nil)

	ctx := bundle.Context(context.Background(), b)
	cmd := NewGenerateDashboardCommand()
	cmd.SetContext(ctx)
	err := cmd.Flag("existing-id").Value.Set("f00dcafe")
	require.NoError(t, err)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Assert the contents of the generated configuration
	data, err := os.ReadFile(filepath.Join(root, "resources", "this_is_a_test_dashboard.dashboard.yml"))
	require.NoError(t, err)
	assert.Equal(t, `resources:
  dashboards:
    this_is_a_test_dashboard:
      display_name: "This is a test dashboard"
      warehouse_id: w4r3h0us3
      file_path: ../src/this_is_a_test_dashboard.lvdash.json
`, string(data))

	data, err = os.ReadFile(filepath.Join(root, "src", "this_is_a_test_dashboard.lvdash.json"))
	require.NoError(t, err)
	assert.JSONEq(t, `{"pages":[{"displayName":"New Page","layout":[],"name":"12345678"}]}`, string(data))
}

func TestDashboard_ExistingID_NotFound(t *testing.T) {
	root := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: root,
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	dashboardsAPI := m.GetMockLakeviewAPI()
	dashboardsAPI.EXPECT().GetByDashboardId(mock.Anything, "f00dcafe").Return(nil, &apierr.APIError{
		StatusCode: 404,
	})

	ctx := bundle.Context(context.Background(), b)
	cmd := NewGenerateDashboardCommand()
	cmd.SetContext(ctx)
	err := cmd.Flag("existing-id").Value.Set("f00dcafe")
	require.NoError(t, err)

	err = cmd.RunE(cmd, []string{})
	require.Error(t, err)
}

func TestDashboard_ExistingPath_Nominal(t *testing.T) {
	root := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: root,
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	workspaceAPI := m.GetMockWorkspaceAPI()
	workspaceAPI.EXPECT().GetStatusByPath(mock.Anything, "/path/to/dashboard").Return(&workspace.ObjectInfo{
		ObjectType: workspace.ObjectTypeDashboard,
		ResourceId: "f00dcafe",
	}, nil)

	dashboardsAPI := m.GetMockLakeviewAPI()
	dashboardsAPI.EXPECT().GetByDashboardId(mock.Anything, "f00dcafe").Return(&dashboards.Dashboard{
		DashboardId:         "f00dcafe",
		DisplayName:         "This is a test dashboard",
		SerializedDashboard: `{"pages":[{"displayName":"New Page","layout":[],"name":"12345678"}]}`,
		WarehouseId:         "w4r3h0us3",
	}, nil)

	ctx := bundle.Context(context.Background(), b)
	cmd := NewGenerateDashboardCommand()
	cmd.SetContext(ctx)
	err := cmd.Flag("existing-path").Value.Set("/path/to/dashboard")
	require.NoError(t, err)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err)

	// Assert the contents of the generated configuration
	data, err := os.ReadFile(filepath.Join(root, "resources", "this_is_a_test_dashboard.dashboard.yml"))
	require.NoError(t, err)
	assert.Equal(t, `resources:
  dashboards:
    this_is_a_test_dashboard:
      display_name: "This is a test dashboard"
      warehouse_id: w4r3h0us3
      file_path: ../src/this_is_a_test_dashboard.lvdash.json
`, string(data))

	data, err = os.ReadFile(filepath.Join(root, "src", "this_is_a_test_dashboard.lvdash.json"))
	require.NoError(t, err)
	assert.JSONEq(t, `{"pages":[{"displayName":"New Page","layout":[],"name":"12345678"}]}`, string(data))
}

func TestDashboard_ExistingPath_NotFound(t *testing.T) {
	root := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: root,
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	workspaceAPI := m.GetMockWorkspaceAPI()
	workspaceAPI.EXPECT().GetStatusByPath(mock.Anything, "/path/to/dashboard").Return(nil, &apierr.APIError{
		StatusCode: 404,
	})

	ctx := bundle.Context(context.Background(), b)
	cmd := NewGenerateDashboardCommand()
	cmd.SetContext(ctx)
	err := cmd.Flag("existing-path").Value.Set("/path/to/dashboard")
	require.NoError(t, err)

	err = cmd.RunE(cmd, []string{})
	require.Error(t, err)
}
