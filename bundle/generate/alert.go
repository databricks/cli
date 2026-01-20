package generate

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

func ConvertAlertToValue(alert *sql.AlertV2, filePath string) (dyn.Value, error) {
	// The majority of fields of the alert struct are present in .dbalert.json file.
	// We copy the relevant fields manually.
	dv := map[string]dyn.Value{
		"display_name": dyn.NewValue(alert.DisplayName, []dyn.Location{{Line: 1}}),
		"warehouse_id": dyn.NewValue(alert.WarehouseId, []dyn.Location{{Line: 2}}),
		"file_path":    dyn.NewValue(filePath, []dyn.Location{{Line: 3}}),
	}

	return dyn.V(dv), nil
}
