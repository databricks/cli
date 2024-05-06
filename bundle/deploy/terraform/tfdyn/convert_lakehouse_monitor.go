package tfdyn

import (
	"context"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

func convertLakehouseMonitorResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceLakehouseMonitor{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "lakehouse monitor normalization diagnostic: %s", diag.Summary)
	}
	return vout, nil
}

type lakehouseMonitorConverter struct{}

func (lakehouseMonitorConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertLakehouseMonitorResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.LakehouseMonitor[key] = vout.AsAny()

	return nil
}

func init() {
	registerConverter("lakehouse_monitors", lakehouseMonitorConverter{})
}
