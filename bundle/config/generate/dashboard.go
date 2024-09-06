package generate

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

func ConvertDashboardToValue(dashboard *dashboards.Dashboard, filePath string) (dyn.Value, error) {
	// The majority of fields of the dashboard struct are read-only.
	// We copy the relevant fields manually.
	dv := map[string]dyn.Value{
		"display_name":    dyn.NewValue(dashboard.DisplayName, []dyn.Location{{Line: 1}}),
		"parent_path":     dyn.NewValue("${workspace.file_path}", []dyn.Location{{Line: 2}}),
		"warehouse_id":    dyn.NewValue(dashboard.WarehouseId, []dyn.Location{{Line: 3}}),
		"definition_path": dyn.NewValue(filePath, []dyn.Location{{Line: 4}}),
	}

	return dyn.V(dv), nil
}
