package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

func convertMonitorResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceLakehouseMonitor{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "monitor normalization diagnostic: %s", diag.Summary)
	}
	return vout, nil
}

type monitorConverter struct{}

func (monitorConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertMonitorResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.LakehouseMonitor[key] = vout.AsAny()

	return nil
}

func init() {
	registerConverter("monitors", monitorConverter{})
}
