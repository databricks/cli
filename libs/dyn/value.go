package dyn

import (
	"fmt"
	"sort"
	"time"

	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v3"
)

type Value struct {
	v any

	k Kind
	l Location

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

func (v Value) AsMap() (map[string]Value, bool) {
	m, ok := v.v.(map[string]Value)
	return m, ok
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

func (v Value) MarshalYAML() (interface{}, error) {
	switch v.Kind() {
	case KindMap:
		m, _ := v.AsMap()
		keys := maps.Keys(m)
		// We're using location lines to define the order of keys in YAML.
		// The location is set when we convert API response struct to config.Value representation
		// See convert.convertMap for details
		sort.SliceStable(keys, func(i, j int) bool {
			return m[keys[i]].Location().Line < m[keys[j]].Location().Line
		})

		content := make([]*yaml.Node, 0)
		for _, k := range keys {
			item := m[k]
			node := yaml.Node{Kind: yaml.ScalarNode, Value: k}
			in, err := item.MarshalYAML()
			c := in.(*yaml.Node)
			if err != nil {
				return nil, err
			}
			content = append(content, &node)
			content = append(content, c)
		}

		return &yaml.Node{Kind: yaml.MappingNode, Content: content}, nil
	case KindSequence:
		s, _ := v.AsSequence()
		content := make([]*yaml.Node, 0)
		for _, item := range s {
			in, err := item.MarshalYAML()
			c := in.(*yaml.Node)
			if err != nil {
				return nil, err
			}
			content = append(content, c)
		}
		return &yaml.Node{Kind: yaml.SequenceNode, Content: content}, nil
	case KindNil:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: "null"}, nil
	case KindString:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v.MustString()}, nil
	case KindBool:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(v.MustBool())}, nil
	case KindInt:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(v.MustInt())}, nil
	case KindFloat:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: fmt.Sprint(v.MustFloat())}, nil
	case KindTime:
		return &yaml.Node{Kind: yaml.ScalarNode, Value: v.MustTime().UTC().String()}, nil
	default:
		// Panic because we only want to deal with known types.
		panic(fmt.Sprintf("invalid kind: %d", v.k))
	}
}

func (v *Value) SetLocation(l Location) {
	v.l = l
}
