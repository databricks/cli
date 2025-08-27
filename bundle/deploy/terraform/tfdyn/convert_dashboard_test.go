package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertDashboard(t *testing.T) {
	src := resources.Dashboard{
		DashboardConfig: resources.DashboardConfig{
			Dashboard: dashboards.Dashboard{
				DisplayName: "my dashboard",
				WarehouseId: "f00dcafe",
				ParentPath:  "/some/path",
			},
			EmbedCredentials: true,
		},

		Permissions: []resources.DashboardPermission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = dashboardConverter{}.Convert(ctx, "my_dashboard", vin, out)
	require.NoError(t, err)

	// Assert equality on the job
	assert.Equal(t, map[string]any{
		"display_name":      "my dashboard",
		"warehouse_id":      "f00dcafe",
		"parent_path":       "/some/path",
		"embed_credentials": true,
	}, out.Dashboard["my_dashboard"])

	// Assert equality on the permissions
	assert.Equal(t, &schema.ResourcePermissions{
		DashboardId: "${databricks_dashboard.my_dashboard.id}",
		AccessControl: []schema.ResourcePermissionsAccessControl{
			{
				PermissionLevel: "CAN_VIEW",
				UserName:        "jane@doe.com",
			},
		},
	}, out.Permissions["dashboard_my_dashboard"])
}

func TestConvertDashboardSerializedDashboardString(t *testing.T) {
	src := resources.Dashboard{
		DashboardConfig: resources.DashboardConfig{
			SerializedDashboard: `{ "json": true }`,
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = dashboardConverter{}.Convert(ctx, "my_dashboard", vin, out)
	require.NoError(t, err)

	// Assert that the "serialized_dashboard" is included.
	assert.Subset(t, out.Dashboard["my_dashboard"], map[string]any{
		"serialized_dashboard": `{ "json": true }`,
	})
}

func TestConvertDashboardSerializedDashboardAny(t *testing.T) {
	src := resources.Dashboard{
		DashboardConfig: resources.DashboardConfig{
			SerializedDashboard: map[string]any{
				"pages": []map[string]any{
					{
						"displayName": "New Page",
						"layout":      []map[string]any{},
					},
				},
			},
		},
		FilePath: "some/path/to/dashboard.lvdash.json",
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = dashboardConverter{}.Convert(ctx, "my_dashboard", vin, out)
	require.NoError(t, err)

	// Assert that the "serialized_dashboard" is included.
	assert.Subset(t, out.Dashboard["my_dashboard"], map[string]any{
		"serialized_dashboard": `{"pages":[{"displayName":"New Page","layout":[]}]}`,
	})

	// Assert that the "file_path" is dropped.
	assert.NotContains(t, out.Dashboard["my_dashboard"], "file_path")
}
