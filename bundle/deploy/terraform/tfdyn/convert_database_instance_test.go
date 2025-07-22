package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertDatabaseInstance(t *testing.T) {
	src := resources.DatabaseInstance{
		DatabaseInstance: database.DatabaseInstance{
			Name:                      "test-db-instance",
			Capacity:                  "CU_4",
			NodeCount:                 2,
			EnableReadableSecondaries: true,
			RetentionWindowInDays:     14,
			Stopped:                   false,
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = databaseInstanceConverter{}.Convert(ctx, "my_database_instance", vin, out)
	require.NoError(t, err)

	databaseInstance := out.DatabaseInstance["my_database_instance"]
	assert.Equal(t, map[string]any{
		"name":                        "test-db-instance",
		"capacity":                    "CU_4",
		"node_count":                  int64(2),
		"enable_readable_secondaries": true,
		"retention_window_in_days":    int64(14),
	}, databaseInstance)
}

func TestConvertDatabaseInstanceWithMinimalConfig(t *testing.T) {
	src := resources.DatabaseInstance{
		DatabaseInstance: database.DatabaseInstance{
			Name:     "minimal-db-instance",
			Capacity: "CU_1",
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = databaseInstanceConverter{}.Convert(ctx, "minimal_database_instance", vin, out)
	require.NoError(t, err)

	databaseInstance := out.DatabaseInstance["minimal_database_instance"]
	assert.Equal(t, map[string]any{
		"name":     "minimal-db-instance",
		"capacity": "CU_1",
	}, databaseInstance)
}

func TestConvertDatabaseInstanceWithPermissions(t *testing.T) {
	src := resources.DatabaseInstance{
		DatabaseInstance: database.DatabaseInstance{
			Name:     "db-instance-with-permissions",
			Capacity: "CU_2",
		},
		Permissions: []resources.DatabaseInstancePermission{
			{
				Level:    "CAN_USE",
				UserName: "user@example.com",
			},
			{
				Level:                "CAN_MANAGE",
				ServicePrincipalName: "sp-name",
			},
		},
	}

	// Add permissions to the dynamic value
	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = databaseInstanceConverter{}.Convert(ctx, "db_with_permissions", vin, out)
	require.NoError(t, err)

	databaseInstance := out.DatabaseInstance["db_with_permissions"]
	assert.Equal(t, map[string]any{
		"name":     "db-instance-with-permissions",
		"capacity": "CU_2",
	}, databaseInstance)

	// Assert permissions were created
	assert.Equal(t, &schema.ResourcePermissions{
		DatabaseInstanceName: "${databricks_database_instance.db_with_permissions.name}",
		AccessControl: []schema.ResourcePermissionsAccessControl{
			{
				PermissionLevel: "CAN_USE",
				UserName:        "user@example.com",
			},
			{
				PermissionLevel:      "CAN_MANAGE",
				ServicePrincipalName: "sp-name",
			},
		},
	}, out.Permissions["database_instance_db_with_permissions"])
}
