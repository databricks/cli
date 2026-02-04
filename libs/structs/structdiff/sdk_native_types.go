package structdiff

import (
	"encoding/json"
	"reflect"
	"slices"

	sdkduration "github.com/databricks/databricks-sdk-go/common/types/duration"
	sdkfieldmask "github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
)

// sdkNativeTypes is a list of SDK native types that use custom JSON marshaling
// and should be compared using their string representation.
var sdkNativeTypes = []reflect.Type{
	reflect.TypeFor[sdkduration.Duration](),
	reflect.TypeFor[sdktime.Time](),
	reflect.TypeFor[sdkfieldmask.FieldMask](),
}

// isSDKNativeType returns true if t is an SDK native type.
func isSDKNativeType(t reflect.Type) bool {
	return slices.Contains(sdkNativeTypes, t)
}

// marshalSDKNative converts an SDK native type to its string representation.
func marshalSDKNative(v reflect.Value) (string, error) {
	jsonBytes, err := json.Marshal(v.Interface())
	if err != nil {
		return "", err
	}
	var str string
	if err := json.Unmarshal(jsonBytes, &str); err != nil {
		return "", err
	}
	return str, nil
}
