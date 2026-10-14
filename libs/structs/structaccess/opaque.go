package structaccess

import (
	"encoding/json"
	"reflect"
	"sync"
)

var (
	jsonMarshalerType = reflect.TypeFor[json.Marshaler]()

	// Keyed by reflect.Type; IsOpaqueStruct is called for every struct visited
	// during a diff or walk, so the field scan is done once per type.
	opaqueStructCache sync.Map
)

// sdkTypesPkgPath is the package of the sdk-go native types: types.Time,
// types.Duration and types.FieldMask[T]. They marshal to JSON strings, so their
// Go fields (e.g. Time.Seconds) are not paths in the JSON representation.
const sdkTypesPkgPath = "github.com/databricks/sdk-go/core/types"

// IsOpaqueStruct reports whether t is a struct that must be treated as a scalar
// rather than walked field by field: the sdk-go native types, and any struct
// whose fields cannot be walked (no exported fields) but that still carries a
// value, marshaling itself through json.Marshaler. Without this, any two of
// their values would compare equal or be reported by their Go fields.
//
// Genuinely empty proto messages (jobs.ScheduleTriggerState, and similar) have
// no exported fields either, but they implement no marshaler and hold nothing,
// so they are not opaque and keep comparing equal.
func IsOpaqueStruct(t reflect.Type) bool {
	if cached, ok := opaqueStructCache.Load(t); ok {
		return cached.(bool)
	}
	result := isOpaqueStruct(t)
	opaqueStructCache.Store(t, result)
	return result
}

func isOpaqueStruct(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return false
	}
	if t.PkgPath() == sdkTypesPkgPath {
		return true
	}
	for sf := range t.Fields() {
		if sf.IsExported() {
			return false
		}
	}
	return t.Implements(jsonMarshalerType) || reflect.PointerTo(t).Implements(jsonMarshalerType)
}
