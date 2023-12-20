package config

import (
	"fmt"
	"maps"
	"slices"
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

// InvalidValue is equal to the zero-value of Value.
var InvalidValue = Value{
	k: KindInvalid,
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

func (v Value) set(prefix, suffix Path, value Value) (Value, error) {
	var err error

	if len(suffix) == 0 {
		return value, nil
	}

	prefix = prefix.Append(suffix[0])

	// Pick first component.
	pc := suffix[0]
	switch v.k {
	case KindMap:
		// Expect a key to be set if this is a map.
		if len(pc.key) == 0 {
			return InvalidValue, fmt.Errorf("expected a key index at %s", prefix)
		}

		m := maps.Clone(v.MustMap())
		m[pc.key], err = v.set(prefix, suffix[1:], value)
		if err != nil {
			return InvalidValue, err
		}

		// Return an updated map value.
		return Value{
			v: m,
			k: KindMap,
			l: v.l,
		}, nil

	case KindSequence:
		// Expect an index to be set if this is a sequence.
		if len(pc.key) > 0 {
			return InvalidValue, fmt.Errorf("expected an index at %s", prefix)
		}

		s := slices.Clone(v.MustSequence())
		if pc.index < 0 || pc.index >= len(s) {
			return InvalidValue, fmt.Errorf("index out of bounds under %s", prefix)
		}
		s[pc.index], err = v.set(prefix, suffix[1:], value)
		if err != nil {
			return InvalidValue, err
		}

		// Return an updated sequence value.
		return Value{
			v: s,
			k: KindSequence,
			l: v.l,
		}, nil

	default:
		return InvalidValue, fmt.Errorf("expected a map or sequence under %s", prefix)
	}
}

func (v Value) Set(p Path, value Value) (Value, error) {
	return v.set(EmptyPath, p, value)
}

func (v Value) SetKey(key string, value Value) Value {
	m, ok := v.AsMap()
	if !ok {
		m = make(map[string]Value)
	} else {
		m = maps.Clone(m)
	}

	m[key] = value

	return Value{
		v: m,
		k: KindMap,
		l: v.l,
	}
}

func (v Value) AsSequence() ([]Value, bool) {
	s, ok := v.v.([]Value)
	return s, ok
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
