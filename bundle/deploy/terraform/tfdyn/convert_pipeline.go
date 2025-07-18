package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

func convertPipelineResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Modify top-level keys.
	vout, err := renameKeys(vin, map[string]string{
		"libraries":     "library",
		"clusters":      "cluster",
		"notifications": "notification",
	})
	if err != nil {
		return dyn.InvalidValue, err
	}

	vout, err = dyn.DropKeys(vout, []string{"dry_run"})
	if err != nil {
		return dyn.InvalidValue, err
	}

	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourcePipeline{}, vout)
	for _, diag := range diags {
		log.Debugf(ctx, "pipeline normalization diagnostic: %s", diag.Summary)
	}

	return vout, err
}

type pipelineConverter struct{}

func (pipelineConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertPipelineResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.Pipeline[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.PipelineId = fmt.Sprintf("${databricks_pipeline.%s.id}", key)
		out.Permissions["pipeline_"+key] = permissions
	}

	return nil
}

func init() {
	registerConverter("pipelines", pipelineConverter{})
}
