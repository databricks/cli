package generate

import (
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/sdktypes/alerts/v2"
	"github.com/databricks/cli/libs/utils"
)

func ConvertAlertToValue(alert *alerts.Alert, filePath string) (dyn.Value, error) {
	// The majority of fields of the alert struct are present in .dbalert.json file.
	// We copy the relevant fields manually.
	dv := map[string]dyn.Value{
		"display_name": dyn.NewValue(utils.Deref(alert.DisplayName), []diag.Location{{Line: 1}}),
		"warehouse_id": dyn.NewValue(utils.Deref(alert.WarehouseId), []diag.Location{{Line: 2}}),
		"file_path":    dyn.NewValue(filePath, []diag.Location{{Line: 3}}),
	}

	return dyn.V(dv), nil
}
