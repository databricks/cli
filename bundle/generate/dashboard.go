package generate

import (
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/sdktypes/lakeview/v1"
	"github.com/databricks/cli/libs/utils"
)

func ConvertDashboardToValue(dashboard *lakeview.Dashboard, filePath string) (dyn.Value, error) {
	// The majority of fields of the dashboard struct are read-only.
	// We copy the relevant fields manually.
	dv := map[string]dyn.Value{
		"display_name": dyn.NewValue(utils.Deref(dashboard.DisplayName), []diag.Location{{Line: 1}}),
		"warehouse_id": dyn.NewValue(utils.Deref(dashboard.WarehouseId), []diag.Location{{Line: 2}}),
		"file_path":    dyn.NewValue(filePath, []diag.Location{{Line: 3}}),
	}

	return dyn.V(dv), nil
}
