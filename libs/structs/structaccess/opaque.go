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

// IsOpaqueStruct reports whether t is a struct whose fields cannot be walked
// (no exported fields) but that still carries a value, marshaling itself
// through json.Marshaler. The sdk-go core types (types.Time, types.Duration,
// types.FieldMask[T]) are of this shape and are scalars (strings) in JSON.
// Without special handling, any two of their values would compare equal.
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
	for sf := range t.Fields() {
		if sf.IsExported() {
			return false
		}
	}
	return t.Implements(jsonMarshalerType) || reflect.PointerTo(t).Implements(jsonMarshalerType)
}
