package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTestConfig("registered_models", &resources.RegisteredModel{
		Name:            "my_registered_model",
		Comment:         "Test registered model",
		CatalogName:     "main",
		SchemaName:      "default",
		StorageLocation: "s3://my-bucket/my-path",
	})

	registerTestDeps("registered_models.grants", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "registered-model",
			FullName:      "modelid",
			EmbeddedSlice: []catalog.PrivilegeAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeCreateView},
				Principal:  "user@example.com",
			}},
		}, nil
	})
}
