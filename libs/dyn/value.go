package dyn

import (
	"fmt"
	"slices"
)

type Value struct {
	v any

	k Kind

	// List of locations this value is defined at. The first location in the slice
	// is the location returned by the `.Location()` method and is typically used
	// for reporting errors and warnings associated with the value.
	l []Location

	// Whether or not this value is an anchor.
	// If this node doesn't map to a type, we don't need to warn about it.
	anchor bool
}

// InvalidValue is equal to the zero-value of Value.
var InvalidValue = Value{
	k: KindInvalid,
}

// NilValue is a convenient constant for a nil value.
var NilValue = Value{
	k: KindNil,
}

// V constructs a new Value with the given value.
func V(v any) Value {
	return NewValue(v, nil)
}

// NewValue constructs a new Value with the given value and location.
func NewValue(v any, loc []Location) Value {
	switch vin := v.(type) {
	case map[string]Value:
		v = newMappingFromGoMap(vin)
	}

	return Value{
		v: v,
		k: kindOf(v),

		// create a copy of the locations, so that mutations to the original slice
		// don't affect new value.
		l: slices.Clone(loc),
	}
}

// WithLocations returns a new Value with its location set to the given value.
func (v Value) WithLocations(loc []Location) Value {
	return Value{
		v: v.v,
		k: v.k,

		// create a copy of the locations, so that mutations to the original slice
		// don't affect new value.
		l: slices.Clone(loc),
	}
}

func (v Value) AppendLocationsFromValue(w Value) Value {
	return Value{
		v: v.v,
		k: v.k,
		l: append(v.l, w.l...),
	}
}

func (v Value) Kind() Kind {
	return v.k
}

func (v Value) Value() any {
	return v.v
}

func (v Value) Locations() []Location {
	return v.l
}

func (v Value) Location() Location {
	if len(v.l) == 0 {
		return Location{}
	}

	return v.l[0]
}

func (v Value) IsValid() bool {
	return v.k != KindInvalid
}

func (v Value) AsAny() any {
	switch v.k {
	case KindInvalid:
		panic("invoked AsAny on invalid value")
	case KindMap:
		m := v.v.(Mapping)
		out := make(map[string]any, m.Len())
		for _, pair := range m.pairs {
			pk := pair.Key
			pv := pair.Value
			out[pk.MustString()] = pv.AsAny()
		}
		return out
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
		t := v.v.(Time)
		return t.Time()
	default:
		// Panic because we only want to deal with known types.
		panic(fmt.Sprintf("invalid kind: %d", v.k))
	}
}

func (v Value) Get(key string) Value {
	m, ok := v.AsMap()
	if !ok {
		return InvalidValue
	}

	vv, ok := m.GetByString(key)
	if !ok {
		return InvalidValue
	}

	return vv
}

func (v Value) Index(i int) Value {
	s, ok := v.v.([]Value)
	if !ok {
		return InvalidValue
	}

	if i < 0 || i >= len(s) {
		return InvalidValue
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

// eq is an internal only method that compares two values.
// It is used to determine if a value has changed during a visit.
// We need a custom implementation because maps and slices
// cannot be compared with the regular == operator.
func (v Value) eq(w Value) bool {
	if v.k != w.k {
		return false
	}
	if !slices.Equal(v.l, w.l) {
		return false
	}

	switch v.k {
	case KindMap:
		// Compare pointers to the underlying map.
		// This is safe because we don't allow maps to be mutated.
		return &v.v == &w.v
	case KindSequence:
		vs := v.v.([]Value)
		ws := w.v.([]Value)
		lv := len(vs)
		lw := len(ws)
		// If both slices are empty, they are equal.
		if lv == 0 && lw == 0 {
			return true
		}
		// If they have different lengths, they are not equal.
		if lv != lw {
			return false
		}
		// They are both non-empty and have the same length.
		// Compare pointers to the underlying slice.
		// This is safe because we don't allow slices to be mutated.
		return &vs[0] == &ws[0]
	default:
		return v.v == w.v
	}
}
