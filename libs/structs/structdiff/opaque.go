package structdiff

import (
	"bytes"
	"encoding/json"
	"reflect"

	"github.com/databricks/cli/libs/structs/structaccess"
)

// IsOpaqueStruct is [structaccess.IsOpaqueStruct].
func IsOpaqueStruct(t reflect.Type) bool {
	return structaccess.IsOpaqueStruct(t)
}

// equalJSON compares two opaque values through their JSON form, which is the
// only representation that exposes what they hold.
func equalJSON(v1, v2 reflect.Value) (bool, error) {
	b1, err := json.Marshal(v1.Interface())
	if err != nil {
		return false, err
	}
	b2, err := json.Marshal(v2.Interface())
	if err != nil {
		return false, err
	}
	return bytes.Equal(b1, b2), nil
}
