package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

func init() {
	registerTestConfig("models", &resources.MlflowModel{
		Name:        "my_mlflow_model",
		Description: "my_mlflow_model_description",
		Tags: []ml.ModelTag{
			{
				Key:   "k1",
				Value: "v1",
			},
		},
	})

	registerTestDeps("models.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		resp, err := client.ModelRegistry.CreateModel(ctx, ml.CreateModelRequest{
			Name:        "model-permissions",
			Description: "model for permissions testing",
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/registered-models/" + resp.RegisteredModel.Name,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	})

	registerCreateRecoveryOptOut("models", "candidate: model names are unique and the read response has creation_timestamp, but libs/testserver does not set it")
}
