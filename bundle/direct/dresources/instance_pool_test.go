package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
)

func init() {
	registerTestConfig("instance_pools", &resources.InstancePool{
		InstancePoolName: "my-instance-pool",
		NodeTypeId:       "i3.xlarge",
	})

	registerTestDeps("instance_pools.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &PermissionsState{
			ObjectID: "/instance-pools/pool-permissions",
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	})

	registerCreateRecoveryOptOut("instance_pools", "the backend assigns the pool id, and the create request takes no idempotency token")
}
