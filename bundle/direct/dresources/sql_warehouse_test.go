package dresources

import (
	"context"

	"github.com/databricks/databricks-sdk-go"
)

func init() {
	registerWithoutTestFixture("sql_warehouses", "the test server creates a warehouse from an empty request")

	registerTestDeps("sql_warehouses.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &PermissionsState{
			ObjectID: "/sql/warehouses/warehouse-permissions",
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	})

	registerKnownMissingInStateType("sql_warehouses",
		"lifecycle.prevent_destroy",
		"lifecycle.depends_on",
	)

	registerCreateRecoveryOptOut("sql_warehouses", "the backend assigns the warehouse id, and the create request takes no idempotency token")
}
