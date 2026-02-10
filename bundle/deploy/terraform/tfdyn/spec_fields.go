package tfdyn

import (
	"reflect"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// specFieldNames returns the JSON field names from a struct type's top-level fields.
// It is used by postgres resource converters to dynamically determine which fields
// belong in the "spec" block when converting bundle config to Terraform config.
// Using reflection ensures new fields added to the Terraform schema are automatically
// included without manual updates to the converters.
func specFieldNames(v any) []string {
	var names []string
	_ = structwalk.WalkType(reflect.TypeOf(v), func(path *structpath.PatternNode, typ reflect.Type, field *reflect.StructField) bool {
		// Skip root node (path is nil)
		if path == nil {
			return true
		}
		// Collect top-level field names only (depth 1)
		if path.Parent() == nil {
			if key, ok := path.StringKey(); ok {
				names = append(names, key)
			}
		}
		return false
	})
	return names
}
