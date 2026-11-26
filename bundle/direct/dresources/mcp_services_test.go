package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTestConfig("mcp_services", &resources.McpService{
		Parent:       "schemas/main.default",
		McpServiceId: "my_mcp_service",
		Comment:      "Test mcp service",
	})

	registerTestDeps("mcp_services.grants", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "mcp_service",
			FullName:      "main.myschema.mymcpservice",
			EmbeddedSlice: []catalog.PrivilegeAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeApplyTag},
				Principal:  "user@example.com",
			}},
		}, nil
	})
}
