package dyn

import (
	"fmt"
	"sort"
	"strconv"

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

func (v Value) Kind() Kind {
	return v.k
}

func (v Value) Value() any {
	return v.v
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

func (v Value) IsScalarValueInString() bool {
	if v.Kind() != KindString {
		return false
	}

	// Parse value of the string and check if it's a scalar value.
	// If it's a scalar value, we want to quote it.
	switch v.MustString() {
	case "true", "false":
		return true
	default:
		_, err := parseNumber(v.MustString())
		return err == nil
	}
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
		// If the string is a scalar value (bool, int, float and etc.), we want to quote it.
		if v.IsScalarValueInString() {
			return &yaml.Node{Kind: yaml.ScalarNode, Value: v.MustString(), Style: yaml.DoubleQuotedStyle}, nil
		}
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
		panic(fmt.Sprintf("invalid kind: %d", v.Kind()))
	}
}

func parseNumber(s string) (any, error) {
	if i, err := strconv.ParseInt(s, 0, 64); err == nil {
		return i, nil
	}

	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f, nil
	}
	return nil, fmt.Errorf("invalid number: %s", s)
}
