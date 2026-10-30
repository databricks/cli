package generate

import (
	"github.com/databricks/cli/libs/structs/structyaml"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
)

func ConvertGenieSpaceToValue(genieSpace *dashboards.GenieSpace, filePath string) (structyaml.Map, error) {
	// Emit only the fields a user authors in a bundle. serialized_space is
	// written to a separate file and referenced via file_path, and output-only
	// fields (e.g. space_id, etag) must not appear in the generated config, so
	// we build the value field by field rather than marshaling the struct.
	dv := structyaml.M(
		"title", genieSpace.Title,
		"warehouse_id", genieSpace.WarehouseId,
		"file_path", filePath,
	)

	if genieSpace.Description != "" {
		dv.Add("description", genieSpace.Description)
	}

	if genieSpace.ParentPath != "" {
		dv.Add("parent_path", ensureWorkspacePrefix(genieSpace.ParentPath))
	}

	return dv, nil
}
