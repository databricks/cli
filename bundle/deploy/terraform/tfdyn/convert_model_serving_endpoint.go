package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

func convertModelServingEndpointResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceModelServing{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "model serving endpoint normalization diagnostic: %s", diag.Summary)
	}

	return vout, nil
}

type modelServingEndpointConverter struct{}

func (modelServingEndpointConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertModelServingEndpointResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.ModelServing[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.ServingEndpointId = fmt.Sprintf("${databricks_model_serving.%s.serving_endpoint_id}", key)
		out.Permissions["model_serving_"+key] = permissions
	}

	return nil
}

func init() {
	registerConverter("model_serving_endpoints", modelServingEndpointConverter{})
}
