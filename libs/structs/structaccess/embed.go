package structaccess

import (
	"reflect"
)

// EmbeddedSliceFieldName is the Go field name that signals struct walkers to treat
// the field as transparent — its contents appear directly at the parent path level
// without adding the field name to the path. Only supported on slice fields with
// one EmbeddedSlice field per struct.
const EmbeddedSliceFieldName = "EmbeddedSlice"

// findEmbedField returns the value of the EmbeddedSlice field in struct v, if any.
// Returns an invalid reflect.Value if no EmbeddedSlice field exists.
func findEmbedField(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	t := v.Type()
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if sf.Name == EmbeddedSliceFieldName {
			return v.Field(i)
		}
	}
	return reflect.Value{}
}

// findEmbedFieldType returns the type of the EmbeddedSlice field in struct type t, if any.
func findEmbedFieldType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil
	}
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if sf.Name == EmbeddedSliceFieldName {
			return sf.Type
		}
	}
	return nil
}
