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

func convertDashboardResource(ctx context.Context, vin dyn.Value) (dyn.Value, error) {
	var err error

	// Normalize the output value to the target schema.
	vout, diags := convert.Normalize(schema.ResourceDashboard{}, vin)
	for _, diag := range diags {
		log.Debugf(ctx, "dashboard normalization diagnostic: %s", diag.Summary)
	}

	// Include "serialized_dashboard" field if "definition_path" is set.
	if path, ok := vin.Get("definition_path").AsString(); ok {
		vout, err = dyn.Set(vout, "serialized_dashboard", dyn.V(fmt.Sprintf("${file(\"%s\")}", path)))
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to set serialized_dashboard: %w", err)
		}
	}

	// Include marshalled copy of "contents" if set.
	contents := vin.Get("contents")
	if contents.Kind() == dyn.KindMap {
		buf, err := json.Marshal(contents.AsAny())
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to marshal contents: %w", err)
		}
		vout, err = dyn.Set(vout, "serialized_dashboard", dyn.V(string(buf)))
		if err != nil {
			return dyn.InvalidValue, fmt.Errorf("failed to set serialized_dashboard: %w", err)
		}
	}

	return vout, nil
}

type DashboardConverter struct{}

func (DashboardConverter) Convert(ctx context.Context, key string, vin dyn.Value, out *schema.Resources) error {
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
	registerConverter("dashboards", DashboardConverter{})
}
