package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTestConfig("model_services", &resources.ModelService{
		Parent:         "schemas/main.default",
		ModelServiceId: "my_model_service",
		Comment:        "Test model service",
	})

	registerTestDeps("model_services.grants", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "model_service",
			FullName:      "main.myschema.mymodelservice",
			EmbeddedSlice: []catalog.PrivilegeAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeApplyTag},
				Principal:  "user@example.com",
			}},
		}, nil
	})
}
