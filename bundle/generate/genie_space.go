package generate

import (
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/sdktypes/genie/v1"
	"github.com/databricks/cli/libs/utils"
)

func ConvertGenieSpaceToValue(genieSpace *genie.GenieSpace, filePath string) (dyn.Value, error) {
	// Emit only the fields a user authors in a bundle. serialized_space is
	// written to a separate file and referenced via file_path, and output-only
	// fields (e.g. space_id, etag) must not appear in the generated config, so
	// we build the value field by field rather than marshaling the struct.
	dv := map[string]dyn.Value{
		"title":        dyn.NewValue(utils.Deref(genieSpace.Title), []diag.Location{{Line: 1}}),
		"warehouse_id": dyn.NewValue(utils.Deref(genieSpace.WarehouseId), []diag.Location{{Line: 2}}),
		"file_path":    dyn.NewValue(filePath, []diag.Location{{Line: 3}}),
	}

	if description := utils.Deref(genieSpace.Description); description != "" {
		dv["description"] = dyn.NewValue(description, []diag.Location{{Line: 4}})
	}

	if parentPath := utils.Deref(genieSpace.ParentPath); parentPath != "" {
		dv["parent_path"] = dyn.NewValue(ensureWorkspacePrefix(parentPath), []diag.Location{{Line: 5}})
	}

	return dyn.V(dv), nil
}
