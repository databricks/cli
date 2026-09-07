package tfdyn

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

func convertDashboardResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	var err error

	// Normalize the output value to the target schema. ConfigureDashboardSerializedDashboard
	// normalizes serialized_dashboard to a JSON string before this runs, so it maps
	// straight onto the schema's string field.
	vout, diags := convert.Normalize(schema.ResourceDashboard{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "dashboard normalization diagnostic: %s", diag.Summary)
	}

	// Drop the "file_path" field. It's always inlined into "serialized_dashboard".
	vout, err = dyn.DropKeys(vout, []string{"file_path"})
	if err != nil {
		return dyn.InvalidValue, err
	}

	return vout, nil
}

type dashboardConverter struct{}

func (dashboardConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertDashboardResource(ctx, vin)
	if err != nil {
		return err
	}

	vout, err = convertLifecycle(ctx, vout, vin.Get("lifecycle"))
	if err != nil {
		return err
	}

	// Add the converted resource to the output.
	out.Dashboard[key] = vout.AsAny()

	// Configure permissions for this resource.
	if permissions := convertPermissionsResource(ctx, vin); permissions != nil {
		permissions.DashboardId = fmt.Sprintf("${databricks_dashboard.%s.id}", key)
		out.Permissions["dashboard_"+key] = permissions
	}

	return nil
}

func init() {
	registerConverter("dashboards", dashboardConverter{})
}
