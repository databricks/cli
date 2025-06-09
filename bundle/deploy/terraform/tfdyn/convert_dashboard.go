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

	// Marshal "serialized_dashboard" as JSON if it is set in the input but not in the output.
	vout, err = marshalSerializedDashboard(vin, vout)
	if err != nil {
		return dyn.InvalidValue, err
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
