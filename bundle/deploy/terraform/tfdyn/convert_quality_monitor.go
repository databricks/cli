package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

func convertQualityMonitorResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceQualityMonitor{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "monitor normalization diagnostic: %s", diag.Summary)
	}
	return vout, nil
}

type qualityMonitorConverter struct{}

func (qualityMonitorConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertQualityMonitorResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.QualityMonitor[key] = vout.AsAny()

	return nil
}

func init() {
	registerConverter("quality_monitors", qualityMonitorConverter{})
}
