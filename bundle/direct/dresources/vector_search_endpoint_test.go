package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
)

func init() {
	registerTestConfig("vector_search_endpoints", &resources.VectorSearchEndpoint{
		Name:         "my-endpoint",
		EndpointType: vectorsearch.EndpointTypeStandard,
	})

	registerTestDeps("vector_search_endpoints.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		waiter, err := client.VectorSearchEndpoints.CreateEndpoint(ctx, vectorsearch.CreateEndpoint{
			Name:         "vs-endpoint-permissions",
			EndpointType: vectorsearch.EndpointTypeStandard,
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/vector-search-endpoints/" + waiter.Response.Id,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	})

	registerKnownMissingInRemoteType("vector_search_endpoints",
		"usage_policy_id",
	)
}
