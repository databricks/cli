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

	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceDashboard{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "dashboard normalization diagnostic: %s", diag.Summary)
	}

	// Include "serialized_dashboard" field if "file_path" is set.
	// Note: the Terraform resource supports "file_path" natively, but its
	// change detection mechanism doesn't work as expected at the time of writing (Sep 30).
	if path, ok := vout.Get("file_path").AsString(); ok {
		vout, err = dyn.Set(vout, "serialized_dashboard", dyn.V(fmt.Sprintf("${file(\"%s\")}", path)))
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to set serialized_dashboard: %w", err)
		}
		// Drop the "file_path" field. It is mutually exclusive with "serialized_dashboard".
		vout, err = dyn.Walk(vout, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			switch len(p) {
			case 0:
				return v, nil
			case 1:
				if p[0] == dyn.Key("file_path") {
					return v, dyn.ErrDrop
				}
			}

			// Skip everything else.
			return v, dyn.ErrSkip
		})
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to drop file_path: %w", err)
		}
	}

	return vout, nil
}

type dashboardConverter struct{}

func (dashboardConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
	vout, err := convertDashboardResource(ctx, vin)
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
