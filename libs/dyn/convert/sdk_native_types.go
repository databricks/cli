package convert

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/dynvar"
	sdkduration "github.com/databricks/databricks-sdk-go/common/types/duration"
	sdkfieldmask "github.com/databricks/databricks-sdk-go/common/types/fieldmask"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
)

// sdkNativeTypes maps SDK native types that use custom JSON marshaling and
// should be treated as strings in dyn.Value to a description of their string
// format. These types all implement json.Marshaler and json.Unmarshaler interfaces.
var sdkNativeTypes = map[reflect.Type]string{
	reflect.TypeFor[sdkduration.Duration]():   `a duration in seconds (e.g. "300s")`,
	reflect.TypeFor[sdktime.Time]():           `an RFC 3339 timestamp (e.g. "2023-12-25T10:30:00Z")`,
	reflect.TypeFor[sdkfieldmask.FieldMask](): `a comma-separated list of field paths (e.g. "name,age,email")`,
}

// isSDKNativeType reports whether typ is one of [sdkNativeTypes].
func isSDKNativeType(typ reflect.Type) bool {
	_, ok := sdkNativeTypes[typ]
	return ok
}

// fromTypedSDKNative converts SDK native types to dyn.Value.
// SDK native types (duration.Duration, time.Time, fieldmask.FieldMask) use
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

// normalizeSDKNative normalizes an SDK native type as a string and checks that it
// parses, so a malformed value is reported with its location instead of failing
// later in [ToTyped].
func (n normalizeOptions) normalizeSDKNative(typ reflect.Type, src dyn.Value, path dyn.Path) (dyn.Value, diag.Diagnostics) {
	v, diags := n.normalizeString(reflect.TypeFor[string](), src, path)
	if !v.IsValid() || dynvar.IsPureVariableReference(v.MustString()) {
		return v, diags
	}

	if err := toTypedSDKNative(reflect.New(typ).Elem(), v); err != nil {
		return dyn.InvalidValue, diags.Append(diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("cannot parse %q as %s", v.MustString(), sdkNativeTypes[typ]),
			Locations: []dyn.Location{src.Location()},
			Paths:     []dyn.Path{path},
		})
	}
	return v, diags
}

// toTypedSDKNative converts a dyn.Value to an SDK native type.
// SDK native types (duration.Duration, time.Time, fieldmask.FieldMask) use
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
