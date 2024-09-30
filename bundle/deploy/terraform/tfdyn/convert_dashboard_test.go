package tfdyn

import (
	"context"
	"fmt"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertDashboard(t *testing.T) {
	var src = resources.Dashboard{
		DisplayName:      "my dashboard",
		WarehouseID:      "f00dcafe",
		ParentPath:       "/some/path",
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
	var src = resources.Dashboard{
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

func TestConvertDashboardEmbedCredentialsPassthrough(t *testing.T) {
	for _, v := range []bool{true, false} {
		t.Run(fmt.Sprintf("set to %v", v), func(t *testing.T) {
			vin := dyn.V(map[string]dyn.Value{
				"embed_credentials": dyn.V(v),
			})

			ctx := context.Background()
			out := schema.NewResources()
			err := dashboardConverter{}.Convert(ctx, "my_dashboard", vin, out)
			require.NoError(t, err)

			// Assert that the "embed_credentials" is set as configured.
			assert.Subset(t, out.Dashboard["my_dashboard"], map[string]any{
				"embed_credentials": v,
			})
		})
	}
}

func TestConvertDashboardEmbedCredentialsDefault(t *testing.T) {
	vin := dyn.V(map[string]dyn.Value{})

	ctx := context.Background()
	out := schema.NewResources()
	err := dashboardConverter{}.Convert(ctx, "my_dashboard", vin, out)
	require.NoError(t, err)

	// Assert that the "embed_credentials" is set to false (by default).
	assert.Subset(t, out.Dashboard["my_dashboard"], map[string]any{
		"embed_credentials": false,
	})
}
