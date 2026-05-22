package generate

import (
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

func ConvertGenieSpaceToValue(genieSpace *dashboards.GenieSpace, filePath string) (dyn.Value, error) {
	// The majority of fields of the genie space struct are read-only.
	// We copy the relevant fields manually.
	dv := map[string]dyn.Value{
		"title":        dyn.NewValue(genieSpace.Title, []dyn.Location{{Line: 1}}),
		"warehouse_id": dyn.NewValue(genieSpace.WarehouseId, []dyn.Location{{Line: 2}}),
		"file_path":    dyn.NewValue(filePath, []dyn.Location{{Line: 3}}),
	}

	if genieSpace.Description != "" {
		dv["description"] = dyn.NewValue(genieSpace.Description, []dyn.Location{{Line: 4}})
	}

	if genieSpace.ParentPath != "" {
		dv["parent_path"] = dyn.NewValue(genieSpace.ParentPath, []dyn.Location{{Line: 5}})
	}

	return dyn.V(dv), nil
}
