package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertModelServingEndpoint(t *testing.T) {
	src := resources.ModelServingEndpoint{
		CreateServingEndpoint: serving.CreateServingEndpoint{
			Name: "name",
			Config: &serving.EndpointCoreConfigInput{
				ServedModels: []serving.ServedModelInput{
					{
						ModelName:          "model_name",
						ModelVersion:       "1",
						ScaleToZeroEnabled: true,
						WorkloadSize:       "Small",
					},
				},
				TrafficConfig: &serving.TrafficConfig{
					Routes: []serving.Route{
						{
							ServedModelName:   "model_name-1",
							TrafficPercentage: 100,
						},
					},
				},
			},
		},
		Permissions: []resources.ModelServingEndpointPermission{
			{
				Level:    "CAN_VIEW",
				UserName: "jane@doe.com",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = modelServingEndpointConverter{}.Convert(ctx, "my_model_serving_endpoint", vin, out)
	require.NoError(t, err)

	// Assert equality on the model serving endpoint
	assert.Equal(t, map[string]any{
		"name": "name",
		"config": map[string]any{
			"served_models": []any{
				map[string]any{
					"model_name":            "model_name",
					"model_version":         "1",
					"scale_to_zero_enabled": true,
					"workload_size":         "Small",
				},
			},
			"traffic_config": map[string]any{
				"routes": []any{
					map[string]any{
						"served_model_name":  "model_name-1",
						"traffic_percentage": int64(100),
					},
				},
			},
		},
	}, out.ModelServing["my_model_serving_endpoint"])

	// Assert equality on the permissions
	assert.Equal(t, &schema.ResourcePermissions{
		ServingEndpointId: "${databricks_model_serving.my_model_serving_endpoint.serving_endpoint_id}",
		AccessControl: []schema.ResourcePermissionsAccessControl{
			{
				PermissionLevel: "CAN_VIEW",
				UserName:        "jane@doe.com",
			},
		},
	}, out.Permissions["model_serving_my_model_serving_endpoint"])
}
