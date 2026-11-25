package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/database"
)

func init() {
	registerTestConfig("database_instances", &resources.DatabaseInstance{
		Name: "mydbinstance",
	})

	registerTestDeps("database_instances.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		waiter, err := client.Database.CreateDatabaseInstance(ctx, database.CreateDatabaseInstanceRequest{
			DatabaseInstance: database.DatabaseInstance{
				Name: "dbinstance-permissions",
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/database-instances/" + waiter.Response.Name,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	})

	registerCreateRecoveryOptOut("database_instances", "candidate: instance names are unique and the read response has creation_time; not done yet")
}
