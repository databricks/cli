package dynpath

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
)

// ConvertPathNodeToDynPath converts a PathNode to dyn path format string.
// Uses the provided root type to determine context-aware wildcard formatting:
// - BracketStar accessing maps renders as "parent.*"
// - BracketStar accessing arrays/slices renders as "parent[*]"
// - DotStar always renders as "parent.*"
func ConvertPathNodeToDynPath(path *structpath.PathNode, rootType reflect.Type) string {
	segments := path.AsSlice()
	var result strings.Builder
	currentType := rootType

	for _, segment := range segments {
		// Dereference pointers
		for currentType != nil && currentType.Kind() == reflect.Pointer {
			currentType = currentType.Elem()
		}

		if index, ok := segment.Index(); ok {
			// Array/slice index access
			result.WriteString("[")
			result.WriteString(strconv.Itoa(index))
			result.WriteString("]")
			if currentType != nil && (currentType.Kind() == reflect.Array || currentType.Kind() == reflect.Slice) {
				currentType = currentType.Elem()
			} else {
				currentType = nil
			}

		} else if field, ok := segment.Field(); ok {
			// Struct field access
			if result.Len() > 0 {
				result.WriteString(".")
			}
			result.WriteString(field)
			if currentType != nil && currentType.Kind() == reflect.Struct {
				if sf, _, ok := structaccess.FindStructFieldByKeyType(currentType, field); ok {
					currentType = sf.Type
				} else {
					currentType = nil
				}
			} else {
				currentType = nil
			}

		} else if key, ok := segment.MapKey(); ok {
			// Map key access - always use dot notation in DynPath
			if result.Len() > 0 {
				result.WriteString(".")
			}
			result.WriteString(key)
			if currentType != nil && currentType.Kind() == reflect.Map {
				currentType = currentType.Elem()
			} else {
				currentType = nil
			}

		} else if segment.DotStar() {
			// Field wildcard - always uses dot notation
			if result.Len() > 0 {
				result.WriteString(".")
			}
			result.WriteString("*")

			// If it's a map, we can inspect the type; otherwise we cannot since we don't know what field to look into
			if currentType != nil && currentType.Kind() == reflect.Map {
				currentType = currentType.Elem()
			} else {
				currentType = nil
			}

		} else if segment.BracketStar() {
			if currentType != nil && (currentType.Kind() == reflect.Array || currentType.Kind() == reflect.Slice) {
				result.WriteString("[*]")
				currentType = currentType.Elem()
			} else {
				// Map wildcard uses dot notation in DynPath
				if result.Len() > 0 {
					result.WriteString(".")
				}
				result.WriteString("*")

				if currentType != nil && currentType.Kind() == reflect.Map {
					currentType = currentType.Elem()
				} else {
					currentType = nil
				}

				// QQQ return error if we cannot disambiguate?
			}
		}
	}

	return result.String()
}
