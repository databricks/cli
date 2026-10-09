package generate

import (
	"github.com/databricks/cli/libs/structs/structyaml"
	"github.com/databricks/databricks-sdk-go/service/sql"
)

func ConvertAlertToValue(alert *sql.AlertV2, filePath string) (structyaml.Map, error) {
	// The majority of fields of the alert struct are present in .dbalert.json file.
	// We copy the relevant fields manually.
	return structyaml.M(
		"display_name", alert.DisplayName,
		"warehouse_id", alert.WarehouseId,
		"file_path", filePath,
	), nil
}
