package tfdyn

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertModel(t *testing.T) {
	src := resources.MlflowModel{
		CreateModelRequest: ml.CreateModelRequest{
			Name:        "name",
			Description: "description",
			Tags: []ml.ModelTag{
				{
					Key:   "k1",
					Value: "v1",
				},
				{
					Key:   "k2",
					Value: "v2",
				},
			},
		},
		Permissions: []resources.MlflowModelPermission{
			{
				Level:    "CAN_READ",
				UserName: "jane@doe.com",
			},
		},
	}

	vin, err := convert.FromTyped(src, dyn.NilValue)
	require.NoError(t, err)

	ctx := context.Background()
	out := schema.NewResources()
	err = modelConverter{}.Convert(ctx, "my_model", vin, out)
	require.NoError(t, err)

	// Assert equality on the model
	assert.Equal(t, map[string]any{
		"name":        "name",
		"description": "description",
		"tags": []any{
			map[string]any{
				"key":   "k1",
				"value": "v1",
			},
			map[string]any{
				"key":   "k2",
				"value": "v2",
			},
		},
	}, out.MlflowModel["my_model"])

	// Assert equality on the permissions
	assert.Equal(t, &schema.ResourcePermissions{
		RegisteredModelId: "${databricks_mlflow_model.my_model.registered_model_id}",
		AccessControl: []schema.ResourcePermissionsAccessControl{
			{
				PermissionLevel: "CAN_READ",
				UserName:        "jane@doe.com",
			},
		},
	}, out.Permissions["mlflow_model_my_model"])
}
