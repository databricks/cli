package dyn

import (
	"fmt"
)

// AsMap returns the underlying mapping if this value is a map,
// the zero value and false otherwise.
func (v Value) AsMap() (Mapping, bool) {
	vv, ok := v.v.(Mapping)
	return vv, ok
}

// MustMap returns the underlying mapping if this value is a map,
// panics otherwise.
func (v Value) MustMap() Mapping {
	vv, ok := v.AsMap()
	if !ok || v.k != KindMap {
		panic(fmt.Sprintf("expected kind %s, got %s", KindMap, v.k))
	}
	return vv
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
	if !ok || v.k != KindSequence {
		panic(fmt.Sprintf("expected kind %s, got %s", KindSequence, v.k))
	}
	return vv
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
	if !ok || v.k != KindString {
		panic(fmt.Sprintf("expected kind %s, got %s", KindString, v.k))
	}
	return vv
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
	if !ok || v.k != KindBool {
		panic(fmt.Sprintf("expected kind %s, got %s", KindBool, v.k))
	}
	return vv
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
		return vv, true
	default:
		return 0, false
	}
}

// MustInt returns the underlying int if this value is an int,
// panics otherwise.
func (v Value) MustInt() int64 {
	vv, ok := v.AsInt()
	if !ok || v.k != KindInt {
		panic(fmt.Sprintf("expected kind %s, got %s", KindInt, v.k))
	}
	return vv
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
	if !ok || v.k != KindFloat {
		panic(fmt.Sprintf("expected kind %s, got %s", KindFloat, v.k))
	}
	return vv
}

// AsTime returns the underlying time if this value is a time,
// the zero value and false otherwise.
func (v Value) AsTime() (Time, bool) {
	vv, ok := v.v.(Time)
	return vv, ok
}

// MustTime returns the underlying time if this value is a time,
// panics otherwise.
func (v Value) MustTime() Time {
	vv, ok := v.AsTime()
	if !ok || v.k != KindTime {
		panic(fmt.Sprintf("expected kind %s, got %s", KindTime, v.k))
	}
	return vv
}
