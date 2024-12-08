package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

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
	out.Volume[key] = vout.AsAny()

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
