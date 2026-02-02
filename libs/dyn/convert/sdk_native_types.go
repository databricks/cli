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

	// All SDK native types marshal to JSON strings. Unmarshal to get the raw string value.
	// For example: duration.Duration(300s) -> JSON "300s" -> string "300s"
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
		// Marshal the string to create a valid JSON string literal for unmarshaling.
		jsonBytes, err := json.Marshal(src.MustString())
		if err != nil {
			return err
		}
		return json.Unmarshal(jsonBytes, dst.Addr().Interface())
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
