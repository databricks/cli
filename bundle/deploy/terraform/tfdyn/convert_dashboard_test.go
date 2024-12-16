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
		Dashboard: &dashboards.Dashboard{
			DisplayName: "my dashboard",
			WarehouseId: "f00dcafe",
			ParentPath:  "/some/path",
		},

		EmbedCredentials: true,

		Permissions: []resources.Permission{
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

func TestConvertDashboardFilePath(t *testing.T) {
	src := resources.Dashboard{
		FilePath: "some/path",
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = dashboardConverter{}.Convert(ctx, "my_dashboard", vin, out)
	require.NoError(t, err)

	// Assert that the "serialized_dashboard" is included.
	assert.Subset(t, out.Dashboard["my_dashboard"], map[string]any{
		"serialized_dashboard": "${file(\"some/path\")}",
	})

	// Assert that the "file_path" doesn't carry over.
	assert.NotSubset(t, out.Dashboard["my_dashboard"], map[string]any{
		"file_path": "some/path",
	})
}

func TestConvertDashboardFilePathQuoted(t *testing.T) {
	src := resources.Dashboard{
		FilePath: `C:\foo\bar\baz\dashboard.lvdash.json`,
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = dashboardConverter{}.Convert(ctx, "my_dashboard", vin, out)
	require.NoError(t, err)

	// Assert that the "serialized_dashboard" is included.
	assert.Subset(t, out.Dashboard["my_dashboard"], map[string]any{
		"serialized_dashboard": `${file("C:\\foo\\bar\\baz\\dashboard.lvdash.json")}`,
	})

	// Assert that the "file_path" doesn't carry over.
	assert.NotSubset(t, out.Dashboard["my_dashboard"], map[string]any{
		"file_path": `C:\foo\bar\baz\dashboard.lvdash.json`,
	})
}

func TestConvertDashboardSerializedDashboardString(t *testing.T) {
	src := resources.Dashboard{
		SerializedDashboard: `{ "json": true }`,
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
		SerializedDashboard: map[string]any{
			"pages": []map[string]any{
				{
					"displayName": "New Page",
					"layout":      []map[string]any{},
				},
			},
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
		"serialized_dashboard": `{"pages":[{"displayName":"New Page","layout":[]}]}`,
	})
}
