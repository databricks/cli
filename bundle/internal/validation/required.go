package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
	"github.com/databricks/cli/libs/structs/structwalk"
)

// formatSliceToString formats a list of values into string of the form `{value1, value2, ...}`
// representing a Go slice literal.
func formatSliceToString(values []string) string {
	if len(values) == 0 {
		return "{}"
	}

	var quoted []string
	for _, value := range values {
		quoted = append(quoted, fmt.Sprintf("%q", value))
	}

	return "{" + strings.Join(quoted, ", ") + "}"
}

// extractRequiredFields walks through a struct type and returns the required fields
// (JSON names) by the pattern of the struct that has them.
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

		parentPath := path.Parent().String()
		fieldsByPattern[parentPath] = append(fieldsByPattern[parentPath], fieldName)
		return true
	})

	return fieldsByPattern, err
}

// getGroupingKey determines the grouping key for organizing patterns
func getGroupingKey(pattern string) string {
	parts := strings.Split(pattern, ".")

	// Group resources by their resource type (e.g., "resources.jobs")
	if parts[0] == "resources" && len(parts) > 1 {
		return parts[0] + "." + parts[1]
	}

	// Use the top level key for other fields
	return parts[0]
}
