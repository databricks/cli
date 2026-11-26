package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/ml"
)

func init() {
	registerTestConfig("experiments", &resources.MlflowExperiment{
		Name: "my-experiment",
		Tags: []ml.ExperimentTag{
			{
				Key:   "my-tag",
				Value: "my-value",
			},
		},
		ArtifactLocation: "s3://my-bucket/my-experiment",
	})

	registerTestDeps("experiments.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		resp, err := client.Experiments.CreateExperiment(ctx, ml.CreateExperiment{
			Name: "experiment-permissions",
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/experiments/" + resp.ExperimentId,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	})

	registerCreateRecoveryOptOut("experiments", "candidate: experiment names are unique and the read response has creation_time, but libs/testserver has neither get-by-name nor creation_time")
}
