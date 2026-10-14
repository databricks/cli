package convert

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
)

// sdkTypesPkgPath is the package of the SDK native types: types.Time (RFC3339
// timestamp, e.g. "2023-12-25T10:30:00Z"), types.Duration (protobuf duration,
// e.g. "300s") and types.FieldMask[T] (comma-separated paths). They use custom
// JSON marshaling to a string and are treated as strings in dyn.Value.
const sdkTypesPkgPath = "github.com/databricks/sdk-go/core/types"

// isSDKNativeType reports whether typ is one of the SDK native types. FieldMask
// is generic, so its instantiations are matched by package rather than by type.
func isSDKNativeType(typ reflect.Type) bool {
	return typ.Kind() == reflect.Struct && typ.PkgPath() == sdkTypesPkgPath
}

// fromTypedSDKNative converts SDK native types to dyn.Value.
// SDK native types (types.Time, types.Duration, types.FieldMask[T]) use
// custom JSON marshaling with string representations.
func fromTypedSDKNative(src reflect.Value, ref dyn.Value, options ...fromTypedOptions) (dyn.Value, error) {
	// Check that the reference value is compatible or nil.
	switch ref.Kind() {
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(ref.MustString()) {
			return ref, nil
		}
	case dyn.KindNil:
		// Allow nil reference.
	default:
		return dyn.InvalidValue, fmt.Errorf("cannot convert SDK native type to dynamic type %#v", ref.Kind().String())
	}

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
// SDK native types (types.Time, types.Duration, types.FieldMask[T]) use
// custom JSON marshaling with string representations.
func toTypedSDKNative(dst reflect.Value, src dyn.Value) error {
	switch src.Kind() {
	case dyn.KindString:
		// Ignore pure variable references (e.g. ${var.foo}).
		if dynvar.IsPureVariableReference(src.MustString()) {
			dst.SetZero()
			return nil
		}
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
