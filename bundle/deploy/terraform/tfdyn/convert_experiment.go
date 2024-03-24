package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

func convertExperimentResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceMlflowExperiment{}, vin, make(map[string]string))
	for _, diag := range diags {
		log.Debugf(ctx, "experiment normalization diagnostic: %s", diag.Summary)
	}

	return vout, nil
}

type experimentConverter struct{}

func (experimentConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertExperimentResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.MlflowExperiment[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.ExperimentId = fmt.Sprintf("${databricks_mlflow_experiment.%s.id}", key)
		out.Permissions["mlflow_experiment_"+key] = permissions
	}

	return nil
}

func init() {
	registerConverter("experiments", experimentConverter{})
}
