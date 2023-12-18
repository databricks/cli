package config

import (
	"fmt"
	"time"
)

type Value struct {
	v any

	k Kind
	l Location

	// Whether or not this value is an anchor.
	// If this node doesn't map to a type, we don't need to warn about it.
	anchor bool
}

// NilValue is equal to the zero-value of Value.
var NilValue = Value{
	k: KindNil,
}

// V constructs a new Value with the given value.
func V(v any) Value {
	return Value{
		v: v,
		k: kindOf(v),
	}
}

// NewValue constructs a new Value with the given value and location.
func NewValue(v any, loc Location) Value {
	return Value{
		v: v,
		k: kindOf(v),
		l: loc,
	}
}

// WithLocation returns a new Value with its location set to the given value.
func (v Value) WithLocation(loc Location) Value {
	return Value{
		v: v.v,
		k: v.k,
		l: loc,
	}
}

func (v Value) AsMap() (map[string]Value, bool) {
	m, ok := v.v.(map[string]Value)
	return m, ok
}

func (v Value) Kind() Kind {
	return v.k
}

func (v Value) Location() Location {
	return v.l
}

func (v Value) IsValid() bool {
	return v.k != KindInvalid
}

func (v Value) AsAny() any {
	switch v.k {
	case KindInvalid:
		panic("invoked AsAny on invalid value")
	case KindMap:
		vv := v.v.(map[string]Value)
		m := make(map[string]any, len(vv))
		for k, v := range vv {
			m[k] = v.AsAny()
		}
		return m
	case KindSequence:
		vv := v.v.([]Value)
		a := make([]any, len(vv))
		for i, v := range vv {
			a[i] = v.AsAny()
		}
		return a
	case KindNil:
		return v.v
	case KindString:
		return v.v
	case KindBool:
		return v.v
	case KindInt:
		return v.v
	case KindFloat:
		return v.v
	case KindTime:
		return v.v
	default:
		// Panic because we only want to deal with known types.
		panic(fmt.Sprintf("invalid kind: %d", v.k))
	}
}

func (v Value) Get(key string) Value {
	m, ok := v.AsMap()
	if !ok {
		return NilValue
	}

	vv, ok := m[key]
	if !ok {
		return NilValue
	}

	return vv
}

func (v Value) Index(i int) Value {
	s, ok := v.v.([]Value)
	if !ok {
		return NilValue
	}

	if i < 0 || i >= len(s) {
		return NilValue
	}

	return s[i]
}

func (v Value) MarkAnchor() Value {
	return Value{
		v: v.v,
		k: v.k,
		l: v.l,

		anchor: true,
	}
}

func (v Value) IsAnchor() bool {
	return v.anchor
}

func (v Value) MustMap() map[string]Value {
	return v.v.(map[string]Value)
}

func (v Value) MustSequence() []Value {
	return v.v.([]Value)
}

func (v Value) MustString() string {
	return v.v.(string)
}

func (v Value) MustBool() bool {
	return v.v.(bool)
}

func (v Value) MustInt() int64 {
	switch vv := v.v.(type) {
	case int:
		return int64(vv)
	case int32:
		return int64(vv)
	case int64:
		return int64(vv)
	default:
		panic("not an int")
	}
}

func (v Value) MustFloat() float64 {
	switch vv := v.v.(type) {
	case float32:
		return float64(vv)
	case float64:
		return float64(vv)
	default:
		panic("not a float")
	}
}

func (v Value) MustTime() time.Time {
	return v.v.(time.Time)
}
