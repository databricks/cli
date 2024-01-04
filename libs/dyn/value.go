package dyn

import (
	"fmt"
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
