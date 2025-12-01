package structaccess

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structtag"
)

// GetByString returns the value at the given path inside v.
// This is a convenience function that parses the path string and calls Get.
func GetByString(v any, path string) (any, error) {
	if path == "" {
		return v, nil
	}

	pathNode, err := structpath.Parse(path)
	if err != nil {
		return nil, err
	}

	return Get(v, pathNode)
}

// getValue returns the reflect.Value at the given path inside v.
// This is the internal function that Get() wraps.
func getValue(v any, path *structpath.PathNode) (reflect.Value, error) {
	if path.IsRoot() {
		return reflect.ValueOf(v), nil
	}

	// Convert path to slice for easier iteration
	pathSegments := path.AsSlice()

	cur := reflect.ValueOf(v)
	for _, node := range pathSegments {
		if node.DotStar() || node.BracketStar() {
			return reflect.Value{}, fmt.Errorf("wildcards not supported: %s", path.String())
		}

		var ok bool
		cur, ok = deref(cur)
		if !ok {
			// cannot proceed further due to nil encountered at current location
			return reflect.Value{}, fmt.Errorf("%s: cannot access nil value", node.Parent().String())
		}

		if idx, isIndex := node.Index(); isIndex {
			kind := cur.Kind()
			if kind != reflect.Slice && kind != reflect.Array {
				return reflect.Value{}, fmt.Errorf("%s: cannot index %s", node.String(), kind)
			}
			if idx < 0 || idx >= cur.Len() {
				return reflect.Value{}, fmt.Errorf("%s: index out of range, length is %d", node.String(), cur.Len())
			}
			cur = cur.Index(idx)
			continue
		}

		key, ok := node.StringKey()
		if !ok {
			return reflect.Value{}, errors.New("unsupported path node type")
		}

		nv, err := accessKey(cur, key, node)
		if err != nil {
			return reflect.Value{}, err
		}
		cur = nv
	}

	return cur, nil
}

// Get returns the value at the given path inside v.
// Wildcards ("*" or "[*]") are not supported and return an error.
func Get(v any, path *structpath.PathNode) (any, error) {
	cur, err := getValue(v, path)
	if err != nil {
		return nil, err
	}

	// If the current value is invalid (e.g., omitted due to omitempty), return nil.
	if !cur.IsValid() {
		return nil, nil
	}

	// If the final value is a nil pointer or nil interface, return nil.
	if (cur.Kind() == reflect.Pointer || cur.Kind() == reflect.Interface) && cur.IsNil() {
		return nil, nil
	}

	// Return the resulting value as interface{}; do not force dereference of scalars.
	return cur.Interface(), nil
}

// accessKey returns the field or map entry value selected by key from v.
// v must be non-pointer, non-interface reflect.Value.
func accessKey(v reflect.Value, key string, path *structpath.PathNode) (reflect.Value, error) {
	switch v.Kind() {
	case reflect.Struct:
		// Precalculate ForceSendFields mappings for this struct hierarchy
		forceSendFieldsMap := getForceSendFieldsForFromTyped(v)

		fv, sf, embeddedIndex, ok := findStructFieldByKey(v, key)
		if !ok {
			return reflect.Value{}, fmt.Errorf("%s: field %q not found in %s", path.String(), key, v.Type())
		}

		// Check ForceSendFields using precalculated map
		var force bool
		if fields, exists := forceSendFieldsMap[embeddedIndex]; exists {
			force = containsString(fields, sf.Name)
		}

		// Honor omitempty: if present and value is empty and not forced, treat as omitted (nil).
		jsonTag := structtag.JSONTag(sf.Tag.Get("json"))
		if jsonTag.OmitEmpty() && !force {
			if fv.Kind() == reflect.Pointer {
				if fv.IsNil() {
					return reflect.Value{}, nil
				}
				// Non-nil pointer: check if the pointed-to value is empty for omitempty
				if isEmptyForOmitEmpty(fv.Elem()) {
					return reflect.Value{}, nil
				}
			} else if isEmptyForOmitEmpty(fv) {
				return reflect.Value{}, nil
			}
		}
		return fv, nil
	case reflect.Map:
		kt := v.Type().Key()
		if kt.Kind() != reflect.String {
			return reflect.Value{}, fmt.Errorf("%s: map key must be string, got %s", path.String(), kt)
		}
		mk := reflect.ValueOf(key)
		if kt != mk.Type() {
			mk = mk.Convert(kt)
		}
		mv := v.MapIndex(mk)
		if !mv.IsValid() {
			return reflect.Value{}, fmt.Errorf("%s: key %q not found in map", path.String(), key)
		}
		return mv, nil
	default:
		return reflect.Value{}, fmt.Errorf("%s: cannot access key %q on %s", path.String(), key, v.Kind())
	}
}

// findFieldInStruct searches for a field by JSON key in a single struct (no embedding).
// Returns: fieldValue, structField, found
func findFieldInStruct(v reflect.Value, key string) (reflect.Value, reflect.StructField, bool) {
	t := v.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" { // unexported
			continue
		}
		if sf.Anonymous { // skip embedded fields
			continue
		}

		// Read JSON tag using structtag helper
		name := structtag.JSONTag(sf.Tag.Get("json")).Name()
		if name == "-" {
			name = ""
		}

		if name != "" && name == key {
			// Skip fields marked as internal or readonly via bundle tag
			btag := structtag.BundleTag(sf.Tag.Get("bundle"))
			if btag.Internal() || btag.ReadOnly() {
				continue
			}
			return v.Field(i), sf, true
		}
	}
	return reflect.Value{}, reflect.StructField{}, false
}

// findStructFieldByKey searches exported fields of struct v for a field matching key.
// It matches json tag name (when present and not "-") only.
// It also searches embedded anonymous structs (flattening semantics).
// Returns: fieldValue, structField, embeddedIndex, found
// embeddedIndex is -1 for direct fields, or the index of the embedded struct containing the field.
func findStructFieldByKey(v reflect.Value, key string) (reflect.Value, reflect.StructField, int, bool) {
	t := v.Type()

	// First pass: direct fields
	if fv, sf, found := findFieldInStruct(v, key); found {
		return fv, sf, -1, true
	}

	// Second pass: search embedded anonymous structs (flattening semantics)
	for i := range t.NumField() {
		sf := t.Field(i)
		if !sf.Anonymous {
			continue
		}
		fv := v.Field(i)
		// Dereference pointer anonymous structs
		for fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				// Not initialized; can't descend
				break
			}
			fv = fv.Elem()
		}
		if fv.Kind() != reflect.Struct {
			continue
		}
		if out, osf, found := findFieldInStruct(fv, key); found {
			return out, osf, i, true
		}
	}

	return reflect.Value{}, reflect.StructField{}, -1, false
}

// getForceSendFieldsForFromTyped collects ForceSendFields values for FromTyped operations
// Returns map[structKey][]fieldName where structKey is -1 for direct fields, embedded index for embedded fields
func getForceSendFieldsForFromTyped(v reflect.Value) map[int][]string {
	if !v.IsValid() || v.Type().Kind() != reflect.Struct {
		return make(map[int][]string)
	}

	result := make(map[int][]string)

	for i := range v.Type().NumField() {
		field := v.Type().Field(i)
		fieldValue := v.Field(i)

		if field.Name == "ForceSendFields" && !field.Anonymous {
			// Direct ForceSendFields (structKey = -1)
			if fields, ok := fieldValue.Interface().([]string); ok {
				result[-1] = fields
			}
		} else if field.Anonymous {
			// Embedded struct - check for ForceSendFields inside it
			if embeddedStruct := getEmbeddedStructForReading(fieldValue); embeddedStruct.IsValid() {
				if forceSendField := embeddedStruct.FieldByName("ForceSendFields"); forceSendField.IsValid() {
					if fields, ok := forceSendField.Interface().([]string); ok {
						result[i] = fields
					}
				}
			}
		}
	}

	return result
}

// Helper function for reading - doesn't create nil pointers
func getEmbeddedStructForReading(fieldValue reflect.Value) reflect.Value {
	if fieldValue.Kind() == reflect.Pointer {
		if fieldValue.IsNil() {
			return reflect.Value{} // Don't create, just return invalid
		}
		fieldValue = fieldValue.Elem()
	}
	if fieldValue.Kind() == reflect.Struct {
		return fieldValue
	}
	return reflect.Value{}
}

// containsString checks if a slice contains a specific string
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

// isEmptyForOmitEmpty returns true if the value should be omitted by JSON omitempty.
// This matches JSON encoder behavior, which is different from reflect.IsZero() for slices/maps.
func isEmptyForOmitEmpty(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Slice, reflect.Map, reflect.Array:
		return v.Len() == 0
	case reflect.Interface, reflect.Pointer:
		return v.IsNil()
	case reflect.Struct:
		// Pointers to structs are not considered empty if pointer != nil
		// Structs as values are never empty and omitempty on them has no effect.
		return false
	default:
		return v.IsZero()
	}
}

// deref dereferences pointers and interfaces until it reaches a non-pointer, non-interface value.
// Returns ok=false if it encounters a nil pointer/interface.
func deref(v reflect.Value) (reflect.Value, bool) {
	for {
		switch v.Kind() {
		case reflect.Invalid:
			return v, false
		case reflect.Pointer:
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		case reflect.Interface:
			if v.IsNil() {
				return reflect.Value{}, false
			}
			v = v.Elem()
		default:
			return v, true
		}
	}
}
