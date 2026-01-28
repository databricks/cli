package generate

import (
	"encoding/json"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
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

func ConvertAlertToValueWithDefinition(alert *sql.AlertV2, alertJSON []byte) (dyn.Value, error) {
	// Parse the alert JSON into a generic map for inline embedding
	var alertDef map[string]any
	if err := json.Unmarshal(alertJSON, &alertDef); err != nil {
		return dyn.Value{}, err
	}

	// Convert the parsed JSON to a dyn.Value
	defValue, err := convert.FromTyped(alertDef, dyn.NilValue)
	if err != nil {
		return dyn.Value{}, err
	}

	// Create the configuration with embedded definition
	dv := map[string]dyn.Value{
		"display_name": dyn.NewValue(alert.DisplayName, []dyn.Location{{Line: 1}}),
		"warehouse_id": dyn.NewValue(alert.WarehouseId, []dyn.Location{{Line: 2}}),
		"definition":   defValue,
	}

	return dyn.V(dv), nil
}
