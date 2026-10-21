package main

import (
	"reflect"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// extractRequiredFields walks through a struct type and returns the required fields
// (JSON names, in declaration order) by the pattern of their parent.
func extractRequiredFields(typ reflect.Type) (map[string][]string, error) {
	fieldsByPattern := make(map[string][]string)

	err := structwalk.WalkType(typ, func(path *structpath.PatternNode, _ reflect.Type, field *reflect.StructField) bool {
		if path == nil || field == nil {
			return true
		}

		// Fields without a json tag (e.g. anonymous embeds) have no independent
		// JSON key and cannot be required. Continue walking their children.
		rawTag := field.Tag.Get("json")
		if rawTag == "" {
			return true
		}

		// Do not generate required validation code for fields that are internal or readonly.
		bundleTag := structtag.BundleTag(field.Tag.Get("bundle"))
		if bundleTag.Internal() || bundleTag.ReadOnly() {
			return false
		}

		// The "omitempty" tag indicates the field is optional in bundle config.
		jsonTag := structtag.JSONTag(rawTag)
		if jsonTag.OmitEmpty() {
			return true
		}

		fieldName, ok := path.StringKey()
		if !ok {
			return true
		}

		// A struct that shadows an embedded field of the same JSON name (e.g. apps' lifecycle
		// over BaseResource's) is walked twice, so the same field can be reported twice.
		parentPath := path.Parent().String()
		if !slices.Contains(fieldsByPattern[parentPath], fieldName) {
			fieldsByPattern[parentPath] = append(fieldsByPattern[parentPath], fieldName)
		}
		return true
	})

	return fieldsByPattern, err
}
