package tfdyn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle/internal/tf/schema"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/log"
)

const (
	filePathFieldName            = "file_path"
	serializedDashboardFieldName = "serialized_dashboard"
)

// Marshal "serialized_dashboard" as JSON if it is set in the input but not in the output.
func marshalSerializedDashboard(vin, vout dyn.Value) (dyn.Value, error) {
	// Skip if the "serialized_dashboard" field is already set.
	if v := vout.Get(serializedDashboardFieldName); v.IsValid() {
		return vout, nil
	}

	// Skip if the "serialized_dashboard" field on the input is not set.
	v := vin.Get(serializedDashboardFieldName)
	if !v.IsValid() {
		return vout, nil
	}

	// Marshal the "serialized_dashboard" field as JSON.
	data, err := json.Marshal(v.AsAny())
	if err != nil {
		return dyn.InvalidValue, fmt.Errorf("failed to marshal serialized_dashboard: %w", err)
	}

	// Set the "serialized_dashboard" field on the output.
	return dyn.Set(vout, serializedDashboardFieldName, dyn.V(string(data)))
}

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
	if path, ok := vout.Get(filePathFieldName).AsString(); ok {
		vout, err = dyn.Set(vout, serializedDashboardFieldName, dyn.V(fmt.Sprintf("${file(%q)}", path)))
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to set serialized_dashboard: %w", err)
		}
		// Drop the "file_path" field. It is mutually exclusive with "serialized_dashboard".
		vout, err = dyn.Walk(vout, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			switch len(p) {
			case 0:
				return v, nil
			case 1:
				if p[0] == dyn.Key(filePathFieldName) {
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

	// Marshal "serialized_dashboard" as JSON if it is set in the input but not in the output.
	vout, err = marshalSerializedDashboard(vin, vout)
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
