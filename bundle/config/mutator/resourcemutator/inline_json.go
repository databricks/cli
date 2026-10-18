package resourcemutator

import (
	"encoding/json"
	"reflect"
)

// inlineJSON converts the value of a field that holds a JSON document as a string but
// may be authored as inline YAML. It returns the JSON encoding of an inline map and
// true; false if v is not set or already a string; and the kind of v (as reported in
// diagnostics) if it is neither.
func inlineJSON(v any) (string, bool, string, error) {
	if v == nil {
		return "", false, "", nil
	}
	if _, ok := v.(string); ok {
		return "", false, "", nil
	}
	switch reflect.ValueOf(v).Kind() {
	case reflect.Map, reflect.Struct:
		b, err := json.Marshal(v)
		if err != nil {
			return "", false, "", err
		}
		return string(b), true, "", nil
	case reflect.Slice, reflect.Array:
		return "", false, "sequence", nil
	case reflect.Bool:
		return "", false, "bool", nil
	case reflect.Float32, reflect.Float64:
		return "", false, "float", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "", false, "int", nil
	default:
		return "", false, reflect.ValueOf(v).Kind().String(), nil
	}
}
