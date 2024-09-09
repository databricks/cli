package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

// TODO: Articulate the consequences of deleting a UC volume in the prompt message that
// is displayed.
// TODO: What sort of interpolation should be allowed at `artifact_path`? Should it be
// ${volumes.foo.id} or ${volumes.foo.name} or something else?
func convertVolumeResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceVolume{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "volume normalization diagnostic: %s", diag.Summary)
	}

	return vout, nil
}

type volumeConverter struct{}

func (volumeConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertVolumeResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.Schema[key] = vout.AsAny()

	// Configure grants for this resource.
	if grants := convertGrantsResource(ctx, vin); grants != nil {
		grants.Volume = fmt.Sprintf("${databricks_volume.%s.id}", key)
		out.Grants["volume_"+key] = grants
	}

	return nil
}

func init() {
	registerConverter("volumes", volumeConverter{})
}
