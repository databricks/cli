package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
)

func init() {
	registerTestConfig("model_provider_services", &resources.ModelProviderService{
		Parent:                 "schemas/main.default",
		ModelProviderServiceId: "my_model_provider_service",
		Comment:                "Test model provider service",
	})

	registerTestDeps("model_provider_services.grants", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		return &GrantsState{
			SecurableType: "model_provider_service",
			FullName:      "main.myschema.myproviderservice",
			EmbeddedSlice: []catalog.PrivilegeAssignment{{
				Privileges: []catalog.Privilege{catalog.PrivilegeApplyTag},
				Principal:  "user@example.com",
			}},
		}, nil
	})
}
