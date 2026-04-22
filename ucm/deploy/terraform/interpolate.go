package terraform

import (
	"fmt"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// GroupToTerraformName maps ucm resource-group names to the corresponding
// terraform resource type. Interpolate uses this table to rewrite
// ${resources.<group>.<key>.<field>} → ${databricks_<type>.<key>.<field>}
// so terraform's own dependency graph picks up the edge.
var GroupToTerraformName = map[string]string{
	"catalogs":            "databricks_catalog",
	"schemas":             "databricks_schema",
	"grants":              "databricks_grants",
	"storage_credentials": "databricks_storage_credential",
	"external_locations":  "databricks_external_location",
	"volumes":             "databricks_volume",
	"connections":         "databricks_connection",
}

// Interpolate rewrites ucm-path references in a TF JSON tree to terraform
// paths. A reference whose first two path segments are resources.<known-group>
// gets rewritten; any other reference is left as-is (either a user typo,
// which terraform will surface, or a resource kind not yet registered).
func Interpolate(in dyn.Value) (dyn.Value, error) {
	prefix := dyn.NewPath(dyn.Key("resources"))
	return dynvar.Resolve(in, func(path dyn.Path) (dyn.Value, error) {
		if len(path) < 4 || !path.HasPrefix(prefix) {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}
		tfType, ok := GroupToTerraformName[path[1].Key()]
		if !ok {
			return dyn.InvalidValue, dynvar.ErrSkipResolution
		}
		newPath := dyn.NewPath(dyn.Key(tfType)).Append(path[2:]...)
		return dyn.V(fmt.Sprintf("${%s}", newPath.String())), nil
	})
}
