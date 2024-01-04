package dyn

import (
	"fmt"
	"time"
)

// panicOnTypeMismatch is a helper function for the MustZZZ functions in this file.
// We rather panic with a descriptive error message than a generic one.
func panicOnTypeMismatch[T any](v Value, vv T, ok bool, k Kind) T {
	if !ok || v.k != k {
		panic(fmt.Sprintf("expected kind %s, got %s", k, v.k))
	}
	return vv
}

// AsMap returns the underlying map if this value is a map,
// the zero value and false otherwise.
func (v Value) AsMap() (map[string]Value, bool) {
	vv, ok := v.v.(map[string]Value)
	return vv, ok
}

// MustMap returns the underlying map if this value is a map,
// panics otherwise.
func (v Value) MustMap() map[string]Value {
	vv, ok := v.AsMap()
	return panicOnTypeMismatch(v, vv, ok, KindMap)
}

// AsSequence returns the underlying sequence if this value is a sequence,
// the zero value and false otherwise.
func (v Value) AsSequence() ([]Value, bool) {
	vv, ok := v.v.([]Value)
	return vv, ok
}

// MustSequence returns the underlying sequence if this value is a sequence,
// panics otherwise.
func (v Value) MustSequence() []Value {
	vv, ok := v.AsSequence()
	return panicOnTypeMismatch(v, vv, ok, KindSequence)
}

// AsString returns the underlying string if this value is a string,
// the zero value and false otherwise.
func (v Value) AsString() (string, bool) {
	vv, ok := v.v.(string)
	return vv, ok
}

// MustString returns the underlying string if this value is a string,
// panics otherwise.
func (v Value) MustString() string {
	vv, ok := v.AsString()
	return panicOnTypeMismatch(v, vv, ok, KindString)
}

// AsBool returns the underlying bool if this value is a bool,
// the zero value and false otherwise.
func (v Value) AsBool() (bool, bool) {
	vv, ok := v.v.(bool)
	return vv, ok
}

// MustBool returns the underlying bool if this value is a bool,
// panics otherwise.
func (v Value) MustBool() bool {
	vv, ok := v.AsBool()
	return panicOnTypeMismatch(v, vv, ok, KindBool)
}

// AsInt returns the underlying int if this value is an int,
// the zero value and false otherwise.
func (v Value) AsInt() (int64, bool) {
	switch vv := v.v.(type) {
	case int:
		return int64(vv), true
	case int32:
		return int64(vv), true
	case int64:
		return int64(vv), true
	default:
		return 0, false
	}
}

// MustInt returns the underlying int if this value is an int,
// panics otherwise.
func (v Value) MustInt() int64 {
	vv, ok := v.AsInt()
	return panicOnTypeMismatch(v, vv, ok, KindInt)
}

// AsFloat returns the underlying float if this value is a float,
// the zero value and false otherwise.
func (v Value) AsFloat() (float64, bool) {
	switch vv := v.v.(type) {
	case float32:
		return float64(vv), true
	case float64:
		return float64(vv), true
	default:
		return 0, false
	}
}

// MustFloat returns the underlying float if this value is a float,
// panics otherwise.
func (v Value) MustFloat() float64 {
	vv, ok := v.AsFloat()
	return panicOnTypeMismatch(v, vv, ok, KindFloat)
}

// AsTime returns the underlying time if this value is a time,
// the zero value and false otherwise.
func (v Value) AsTime() (time.Time, bool) {
	vv, ok := v.v.(time.Time)
	return vv, ok
}

// MustTime returns the underlying time if this value is a time,
// panics otherwise.
func (v Value) MustTime() time.Time {
	vv, ok := v.AsTime()
	return panicOnTypeMismatch(v, vv, ok, KindTime)
}
