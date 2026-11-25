package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTestConfig("volumes", &resources.Volume{
		CatalogName: "main",
		SchemaName:  "myschema",
		Name:        "myvolume",
		Comment:     "Test volume",
	})

	registerTestDeps("volumes.grants", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "volume",
			FullName:      "main.myschema.myvolume",
			EmbeddedSlice: []catalog.PrivilegeAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeCreateView},
				Principal:  "user@example.com",
			}},
		}, nil
	})
}
