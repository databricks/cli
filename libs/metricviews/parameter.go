package metricviews

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// ParamDefault is the tri-state default of a v1.1 parameter:
//   - Present=false                  -> the "default" key was absent
//   - Present=true, IsNull=true      -> "default: null"
//   - Present=true, Expr="<value>"   -> "default: <value>"
type ParamDefault struct {
	Present bool   `json:"present"`
	IsNull  bool   `json:"is_null"`
	Expr    string `json:"expr"`
}

// ParameterV11 is a v1.1 parameter declaration with a tri-state default.
type ParameterV11 struct {
	Name     string       `json:"name"`
	DataType string       `json:"data_type"`
	Default  ParamDefault `json:"default"`
}

// UnmarshalYAML distinguishes missing/null/value for the default key by walking
// the mapping node, because a *string would collapse missing and null.
func (p *ParameterV11) UnmarshalYAML(node *yaml.Node) error {
	if err := requireYAMLFields(node, "name", "data_type"); err != nil {
		return err
	}
	*p = ParameterV11{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		val := node.Content[i+1]
		for val.Kind == yaml.AliasNode {
			val = val.Alias
		}
		switch key {
		case "name":
			p.Name = val.Value
		case "data_type":
			p.DataType = val.Value
		case "default":
			p.Default.Present = true
			// An explicit null scalar has tag "!!null" (covers `null` and `~`).
			if val.Tag == "!!null" {
				p.Default.IsNull = true
			} else {
				p.Default.Expr = val.Value
			}
		default:
			return fmt.Errorf("unknown parameter field %q", key)
		}
	}
	return nil
}

// MarshalYAML emits the default key only when present, as null or a value.
func (p ParameterV11) MarshalYAML() (any, error) {
	m := &yaml.Node{Kind: yaml.MappingNode}
	add := func(k string, v *yaml.Node) {
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: k}, v)
	}
	add("name", &yaml.Node{Kind: yaml.ScalarNode, Value: p.Name})
	add("data_type", &yaml.Node{Kind: yaml.ScalarNode, Value: p.DataType})
	switch {
	case !p.Default.Present:
		// omit
	case p.Default.IsNull:
		add("default", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"})
	default:
		// Keep YAML keywords such as "null" as string expressions.
		add("default", &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: p.Default.Expr})
	}
	return m, nil
}

// ParameterV10 is a v1.0 parameter; null and absent both decode to nil.
type ParameterV10 struct {
	Name     string  `yaml:"name" json:"name"`
	DataType string  `yaml:"data_type" json:"data_type"`
	Default  *string `yaml:"default,omitempty" json:"default,omitempty"`
}

func (p *ParameterV10) UnmarshalYAML(node *yaml.Node) error {
	if err := requireYAMLFields(node, "name", "data_type"); err != nil {
		return err
	}
	type alias ParameterV10
	return node.Decode((*alias)(p))
}
