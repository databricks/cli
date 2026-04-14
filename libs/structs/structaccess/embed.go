package structaccess

import (
	"reflect"
	"sync"
)

// EmbeddedSliceFieldName is the Go field name that signals struct walkers to treat
// the field as transparent — its contents appear directly at the parent path level
// without adding the field name to the path. Only supported on slice fields with
// one EmbeddedSlice field per struct.
const EmbeddedSliceFieldName = "EmbeddedSlice"

// embedFieldCache maps a struct reflect.Type to the index of its EmbeddedSlice
// field, or -1 if absent. Populated lazily by embedFieldIndex.
var embedFieldCache sync.Map // map[reflect.Type]int

// embedFieldIndex returns the field index of the EmbeddedSlice field in t,
// or -1 if t has no such field. Dereferences pointer types.
func embedFieldIndex(t reflect.Type) int {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return -1
	}
	if cached, ok := embedFieldCache.Load(t); ok {
		return cached.(int)
	}
	idx := -1
	for i := range t.NumField() {
		sf := t.Field(i)
		if sf.PkgPath != "" {
			continue
		}
		if sf.Name == EmbeddedSliceFieldName {
			idx = i
			break
		}
	}
	embedFieldCache.Store(t, idx)
	return idx
}

// findEmbedField returns the value of the EmbeddedSlice field in struct v, if any.
// Returns an invalid reflect.Value if no EmbeddedSlice field exists.
func findEmbedField(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Struct {
		return reflect.Value{}
	}
	idx := embedFieldIndex(v.Type())
	if idx < 0 {
		return reflect.Value{}
	}
	return v.Field(idx)
}

// findEmbedFieldType returns the type of the EmbeddedSlice field in struct type t, if any.
func findEmbedFieldType(t reflect.Type) reflect.Type {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	idx := embedFieldIndex(t)
	if idx < 0 {
		return nil
	}
	return t.Field(idx).Type
}
