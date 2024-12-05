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
	var src = resources.App{
		SourceCodePath: "./app",
		Config: map[string]interface{}{
			"command": []string{"python", "app.py"},
		},
		App: &apps.App{
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
		Permissions: []resources.Permission{
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
	assert.Equal(t, map[string]interface{}{
		"description": "app description",
		"name":        "app_id",
		"resource": []interface{}{
			map[string]interface{}{
				"name": "job1",
				"job": map[string]interface{}{
					"id":         "1234",
					"permission": "CAN_MANAGE_RUN",
				},
			},
			map[string]interface{}{
				"name": "sql1",
				"sql_warehouse": map[string]interface{}{
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
