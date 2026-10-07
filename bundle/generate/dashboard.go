package generate

import (
	"github.com/databricks/cli/libs/structs/structyaml"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

func ConvertDashboardToValue(dashboard *dashboards.Dashboard, filePath string) (structyaml.Map, error) {
	// The majority of fields of the dashboard struct are read-only.
	// We copy the relevant fields manually.
	return structyaml.M(
		"display_name", dashboard.DisplayName,
		"warehouse_id", dashboard.WarehouseId,
		"file_path", filePath,
	), nil
}
