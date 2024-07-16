package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

func convertRegisteredModelResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceRegisteredModel{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "registered model normalization diagnostic: %s", diag.Summary)
	}

	return vout, nil
}

type registeredModelConverter struct{}

func (registeredModelConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertRegisteredModelResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.RegisteredModel[key] = vout.AsAny()

	// Configure grants for this resource.
	if grants := convertGrantsResource(ctx, vin); grants != nil {
		grants.Function = fmt.Sprintf("${databricks_registered_model.%s.id}", key)
		out.Grants["registered_model_"+key] = grants
	}

	return nil
}

func init() {
	registerConverter("registered_models", registeredModelConverter{})
}
