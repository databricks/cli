package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTestConfig("secrets", &resources.Secret{
		CatalogName: "main",
		SchemaName:  "default",
		Name:        "my_secret",
		Value:       "my_secret_value",
		Comment:     "Test secret",
	})

	registerTestDeps("secrets.grants", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "secret",
			FullName:      "main.default.my_secret",
			EmbeddedSlice: []catalog.PrivilegeAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeSelect},
				Principal:  "user@example.com",
			}},
		}, nil
	})
}
