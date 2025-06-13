package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

func convertAlertResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceAlertV2{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "alert normalization diagnostic: %s", diag.Summary)
	}

	return vout, nil
}

type alertConverter struct{}

func (alertConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertAlertResource(ctx, vin)
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.Alert[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.AlertId = fmt.Sprintf("${databricks_alert_v2.%s.id}", key)
		out.Permissions["alert_"+key] = permissions
	}

	return nil
}
