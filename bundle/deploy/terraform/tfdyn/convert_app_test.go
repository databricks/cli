package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/apps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertApp(t *testing.T) {
	src := resources.App{
		SourceCodePath: "./app",
		Config: map[string]any{
			"command": []string{"python", "app.py"},
		},
		App: apps.App{
			Name:        "app_id",
			Description: "app description",
			Resources: []apps.AppResource{
				{
					Name: "job1",
					Job: &apps.AppResourceJob{
						Id:         "1234",
						Permission: "CAN_MANAGE_RUN",
					},
				},
				{
					Name: "sql1",
					SqlWarehouse: &apps.AppResourceSqlWarehouse{
						Id:         "5678",
						Permission: "CAN_USE",
					},
				},
			},
		},
		Permissions: []resources.AppPermission{
			{
				Level:    "CAN_RUN",
				UserName: "jack@gmail.com",
			},
			{
				Level:                "CAN_MANAGE",
				ServicePrincipalName: "sp",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = appConverter{}.Convert(ctx, "my_app", vin, out)
	require.NoError(t, err)

	app := out.App["my_app"]
	assert.Equal(t, map[string]any{
		"description": "app description",
		"name":        "app_id",
		"no_compute":  true,
		"resources": []any{
			map[string]any{
				"name": "job1",
				"job": map[string]any{
					"id":         "1234",
					"permission": "CAN_MANAGE_RUN",
				},
			},
			map[string]any{
				"name": "sql1",
				"sql_warehouse": map[string]any{
					"id":         "5678",
					"permission": "CAN_USE",
				},
			},
		},
	}, app)

	// Assert equality on the permissions
	assert.Equal(t, &schema.ResourcePermissions{
		AppName: "${databricks_app.my_app.name}",
		AccessControl: []schema.ResourcePermissionsAccessControl{
			{
				PermissionLevel: "CAN_RUN",
				UserName:        "jack@gmail.com",
			},
			{
				PermissionLevel:      "CAN_MANAGE",
				ServicePrincipalName: "sp",
			},
		},
	}, out.Permissions["app_my_app"])
}

func TestConvertAppWithNoDescription(t *testing.T) {
	src := resources.App{
		SourceCodePath: "./app",
		Config: map[string]any{
			"command": []string{"python", "app.py"},
		},
		App: apps.App{
			Name: "app_id",
			Resources: []apps.AppResource{
				{
					Name: "job1",
					Job: &apps.AppResourceJob{
						Id:         "1234",
						Permission: "CAN_MANAGE_RUN",
					},
				},
				{
					Name: "sql1",
					SqlWarehouse: &apps.AppResourceSqlWarehouse{
						Id:         "5678",
						Permission: "CAN_USE",
					},
				},
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = appConverter{}.Convert(ctx, "my_app", vin, out)
	require.NoError(t, err)

	app := out.App["my_app"]
	assert.Equal(t, map[string]any{
		"name":        "app_id",
		"description": "", // Due to Apps API always returning a description field, we set it in the output as well to avoid permanent TF drift
		"no_compute":  true,
		"resources": []any{
			map[string]any{
				"name": "job1",
				"job": map[string]any{
					"id":         "1234",
					"permission": "CAN_MANAGE_RUN",
				},
			},
			map[string]any{
				"name": "sql1",
				"sql_warehouse": map[string]any{
					"id":         "5678",
					"permission": "CAN_USE",
				},
			},
		},
	}, app)
}
