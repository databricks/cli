package convert

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/dyn"
	sdkduration "github.com/databricks/databricks-sdk-go/common/types/duration"
	sdkfieldmask "github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
)

// sdkNativeTypes is a list of SDK native types that use custom JSON marshaling
// and should be treated as strings in dyn.Value. These types all implement
// json.Marshaler and json.Unmarshaler interfaces.
var sdkNativeTypes = []reflect.Type{
	reflect.TypeFor[sdkduration.Duration](),   // Protobuf duration format (e.g., "300s")
	reflect.TypeFor[sdktime.Time](),           // RFC3339 timestamp format (e.g., "2023-12-25T10:30:00Z")
	reflect.TypeFor[sdkfieldmask.FieldMask](), // Comma-separated paths (e.g., "name,age,email")
}

// isSDKNativeType checks if the given type is one of the SDK's native types
// that use custom JSON marshaling and should be treated as strings.
func isSDKNativeType(typ reflect.Type) bool {
	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}
	for _, sdkType := range sdkNativeTypes {
		if typ == sdkType {
			return true
		}
	}
	return false
}

// fromTypedSDKNative converts SDK native types to dyn.Value.
// SDK native types (duration.Duration, time.Time, fieldmask.FieldMask) use
// custom JSON marshaling with string representations.
func fromTypedSDKNative(src reflect.Value, ref dyn.Value, options ...fromTypedOptions) (dyn.Value, error) {
	// Check for zero value first.
	if src.IsZero() && !slices.Contains(options, includeZeroValues) {
		return dyn.NilValue, nil
	}

	// Use JSON marshaling since SDK native types implement json.Marshaler.
	jsonBytes, err := json.Marshal(src.Interface())
	if err != nil {
		return dyn.InvalidValue, err
	}

	// The JSON marshaling produces a quoted string, unmarshal to get the raw string.
	var str string
	if err := json.Unmarshal(jsonBytes, &str); err != nil {
		return dyn.InvalidValue, err
	}

	// Handle empty string as zero value.
	if str == "" && !slices.Contains(options, includeZeroValues) {
		return dyn.NilValue, nil
	}

	return dyn.V(str), nil
}

// toTypedSDKNative converts a dyn.Value to an SDK native type.
// SDK native types (duration.Duration, time.Time, fieldmask.FieldMask) use
// custom JSON marshaling with string representations.
func toTypedSDKNative(dst reflect.Value, src dyn.Value) error {
	switch src.Kind() {
	case dyn.KindString:
		// Use JSON unmarshaling since SDK native types implement json.Unmarshaler.
		jsonStr := fmt.Sprintf("%q", src.MustString())
		return json.Unmarshal([]byte(jsonStr), dst.Addr().Interface())
	case dyn.KindNil:
		dst.SetZero()
		return nil
	default:
		// Fall through to the error case.
	}

	return TypeError{
		value: src,
		msg:   fmt.Sprintf("expected a string, found a %s", src.Kind()),
	}
}
