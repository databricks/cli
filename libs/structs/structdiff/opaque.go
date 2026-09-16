package structdiff

import (
	"bytes"
	"encoding/json"
	"reflect"
	"sync"
)

var (
	jsonMarshalerType = reflect.TypeFor[json.Marshaler]()

	// Keyed by reflect.Type; IsOpaqueStruct is called for every struct visited
	// during a diff, so the field scan is done once per type.
	opaqueStructCache sync.Map
)

// IsOpaqueStruct reports whether t is a struct that diffStruct cannot see into
// (no exported fields to walk) but that still carries a value, marshaling
// itself through json.Marshaler. The SDK's duration.Duration and
// common/types/time.Time are of this shape: each holds a single unexported
// protobuf pointer. Without this, any two of their values compare equal.
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
		// ForceSendFields is skipped when walking fields, so a struct carrying
		// only that is still invisible to the walk.
		if sf.IsExported() && sf.Name != "ForceSendFields" {
			return false
		}
	}
	return t.Implements(jsonMarshalerType) || reflect.PointerTo(t).Implements(jsonMarshalerType)
}

// equalJSON compares two opaque values through their JSON form, which is the
// only representation that exposes what they hold.
func equalJSON(v1, v2 reflect.Value) (bool, error) {
	b1, err := json.Marshal(v1.Interface())
	if err != nil {
		return false, err
	}
	b2, err := json.Marshal(v2.Interface())
	if err != nil {
		return false, err
	}
	return bytes.Equal(b1, b2), nil
}
