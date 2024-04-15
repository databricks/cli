package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

func convertModelResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceMlflowModel{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "model normalization diagnostic: %s", diag.Summary)
	}

	return vout, nil
}

type modelConverter struct{}

func (modelConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertModelResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.MlflowModel[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.RegisteredModelId = fmt.Sprintf("${databricks_mlflow_model.%s.registered_model_id}", key)
		out.Permissions["mlflow_model_"+key] = permissions
	}

	return nil
}

func init() {
	registerConverter("models", modelConverter{})
}
