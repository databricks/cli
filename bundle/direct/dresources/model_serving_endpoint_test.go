package dresources

import (
	"context"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/serving"
)

func init() {
	registerTestConfig("model_serving_endpoints", &resources.ModelServingEndpoint{
		Name: "my-endpoint",
		Config: &serving.EndpointCoreConfigInput{
			Name: "my-endpoint",
			//nolint:staticcheck // SA1019: deprecated AutoCaptureConfigInput kept for bundle config compatibility
			AutoCaptureConfig: &serving.AutoCaptureConfigInput{
				CatalogName:     "main",
				SchemaName:      "myschema",
				TableNamePrefix: "my_table",
				Enabled:         true,
				ForceSendFields: nil,
			},
			ServedModels: nil,
			ServedEntities: []serving.ServedEntityInput{
				{
					EntityName:                "entity-name",
					EntityVersion:             "1",
					WorkloadSize:              "Small",
					ScaleToZeroEnabled:        true,
					WorkloadType:              serving.ServingModelWorkloadTypeCpu,
					EnvironmentVars:           map[string]string{"key": "value"},
					InstanceProfileArn:        "arn:aws:iam::123456789012:instance-profile/my-instance-profile",
					MaxProvisionedConcurrency: 10,
					MaxProvisionedThroughput:  100,
					MinProvisionedConcurrency: 1,
					MinProvisionedThroughput:  10,
					Name:                      "entity-name",
					ProvisionedModelUnits:     100,
					ExternalModel:             nil,
					ForceSendFields:           nil,
				},
			},
			TrafficConfig: &serving.TrafficConfig{
				Routes: []serving.Route{
					{
						ServedModelName:   "model-name-1",
						TrafficPercentage: 100,
					},
				},
			},
		},
	})

	registerTestDeps("model_serving_endpoints.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		waiter, err := client.ServingEndpoints.Create(ctx, serving.CreateServingEndpoint{
			Name: "endpoint-permissions",
			Config: &serving.EndpointCoreConfigInput{
				ServedModels: []serving.ServedModelInput{
					{
						ModelName:          "model-name",
						ModelVersion:       "1",
						WorkloadSize:       "Small",
						ScaleToZeroEnabled: true,
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/serving-endpoints/" + waiter.Response.Name,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	})

	registerKnownMissingInRemoteType("model_serving_endpoints",
		"rate_limits",
	)
}
