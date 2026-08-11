package structwalk

import (
	"encoding/json"
	"reflect"

	"github.com/databricks/cli/libs/structs/structtag"
)

// RedactSensitiveFields returns a JSON-encoded copy of v with all string fields
// tagged `bundle:"sensitive"` replaced by [redacted]. The original value is not
// modified. This is used to scrub secrets before persisting state to disk.
//
// Only string-typed fields at any depth are redacted. The function handles
// embedded structs, pointers, slices, and maps by recursively walking the
// reflect type tree before encoding.
func RedactSensitiveFields(v any, redacted string) ([]byte, error) {
	rv := reflect.ValueOf(v)
	clone := deepCloneRedact(rv, redacted)
	return json.Marshal(clone.Interface())
}

// deepCloneRedact returns a deep copy of rv with sensitive string fields replaced.
func deepCloneRedact(rv reflect.Value, redacted string) reflect.Value {
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return reflect.Zero(rv.Type())
		}
		inner := deepCloneRedact(rv.Elem(), redacted)
		ptr := reflect.New(inner.Type())
		ptr.Elem().Set(inner)
		return ptr
	}

	switch rv.Kind() {
	case reflect.Struct:
		clone := reflect.New(rv.Type()).Elem()
		t := rv.Type()
		for i := range t.NumField() {
			sf := t.Field(i)
			src := rv.Field(i)
			dst := clone.Field(i)

			if !sf.IsExported() {
				continue
			}

			btag := structtag.BundleTag(sf.Tag.Get("bundle"))
			if btag.Sensitive() && sf.Type.Kind() == reflect.String {
				if src.String() != "" {
					dst.SetString(redacted)
				}
				// empty sensitive string stays empty
				continue
			}

			dst.Set(deepCloneRedact(src, redacted))
		}
		return clone

	case reflect.Slice:
		if rv.IsNil() {
			return reflect.Zero(rv.Type())
		}
		clone := reflect.MakeSlice(rv.Type(), rv.Len(), rv.Len())
		for i := range rv.Len() {
			clone.Index(i).Set(deepCloneRedact(rv.Index(i), redacted))
		}
		return clone

	case reflect.Map:
		if rv.IsNil() {
			return reflect.Zero(rv.Type())
		}
		clone := reflect.MakeMap(rv.Type())
		for _, k := range rv.MapKeys() {
			clone.SetMapIndex(k, deepCloneRedact(rv.MapIndex(k), redacted))
		}
		return clone

	default:
		// Scalar, interface, etc. — copy as-is.
		clone := reflect.New(rv.Type()).Elem()
		clone.Set(rv)
		return clone
	}
}
