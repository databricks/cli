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

func TestConvertExperiment(t *testing.T) {
	src := resources.MlflowExperiment{
		Experiment: ml.Experiment{
			Name: "name",
		},
		Permissions: []resources.MlflowExperimentPermission{
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
	err = experimentConverter{}.Convert(ctx, "my_experiment", vin, out)
	require.NoError(t, err)

	// Assert equality on the experiment
	assert.Equal(t, map[string]any{
		"name": "name",
	}, out.MlflowExperiment["my_experiment"])

	// Assert equality on the permissions
	assert.Equal(t, &schema.ResourcePermissions{
		ExperimentId: "${databricks_mlflow_experiment.my_experiment.id}",
		AccessControl: []schema.ResourcePermissionsAccessControl{
			{
				PermissionLevel: "CAN_READ",
				UserName:        "jane@doe.com",
			},
		},
	}, out.Permissions["mlflow_experiment_my_experiment"])
}
