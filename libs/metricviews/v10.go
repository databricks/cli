package metricviews

import "go.yaml.in/yaml/v3"

// ColumnV10 is a v1.0 column; name and expr are required.
type ColumnV10 struct {
	Name        string        `yaml:"name" json:"name"`
	Expr        string        `yaml:"expr" json:"expr"`
	Window      []WindowSpec  `yaml:"window,omitempty" json:"window,omitempty"`
	Comment     *string       `yaml:"comment,omitempty" json:"comment,omitempty"`
	DisplayName *string       `yaml:"display_name,omitempty" json:"display_name,omitempty"`
	Format      *ColumnFormat `yaml:"format,omitempty" json:"format,omitempty"`
	Synonyms    []string      `yaml:"synonyms,omitempty" json:"synonyms,omitempty"`
}

// MetricViewV10 is the v0.1 / v1.0 single-source metric view shape.
type MetricViewV10 struct {
	Version         string           `yaml:"version" json:"version"`
	Source          string           `yaml:"source" json:"source"`
	Filter          *string          `yaml:"filter,omitempty" json:"filter,omitempty"`
	Dimensions      []ColumnV10      `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	Measures        []ColumnV10      `yaml:"measures,omitempty" json:"measures,omitempty"`
	Joins           []Join           `yaml:"joins,omitempty" json:"joins,omitempty"`
	Materialization *Materialization `yaml:"materialization,omitempty" json:"materialization,omitempty"`
	Parameters      []ParameterV10   `yaml:"parameters,omitempty" json:"parameters,omitempty"`
}

// UnmarshalYAML accepts "fields" as an alias for "dimensions".
func (v *MetricViewV10) UnmarshalYAML(node *yaml.Node) error {
	// The local alias type strips this UnmarshalYAML method, so decoding the
	// inlined struct does not recurse. The sibling Fields field carries the
	// "fields" alias alongside the inlined "dimensions".
	type alias MetricViewV10
	aux := struct {
		alias  `yaml:",inline"`
		Fields []ColumnV10 `yaml:"fields"`
	}{}
	if err := node.Decode(&aux); err != nil {
		return err
	}
	*v = MetricViewV10(aux.alias)
	if len(v.Dimensions) == 0 {
		v.Dimensions = aux.Fields
	}
	return nil
}
