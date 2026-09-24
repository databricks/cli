package metricviews

import "go.yaml.in/yaml/v3"

// ColumnV11 is a dimension or measure column in v1.1. An absent Name marks a
// wildcard column (expr is `source.*` / `qualifier.* EXCEPT (...)`).
type ColumnV11 struct {
	Name        *string       `yaml:"name,omitempty" json:"name,omitempty"`
	Expr        string        `yaml:"expr" json:"expr"`
	Window      []WindowSpec  `yaml:"window,omitempty" json:"window,omitempty"`
	Comment     *string       `yaml:"comment,omitempty" json:"comment,omitempty"`
	DisplayName *string       `yaml:"display_name,omitempty" json:"display_name,omitempty"`
	Format      *ColumnFormat `yaml:"format,omitempty" json:"format,omitempty"`
	Synonyms    []string      `yaml:"synonyms,omitempty" json:"synonyms,omitempty"`
}

// Relationship links a source to another source in a multi-source view.
type Relationship struct {
	RefSource  string   `yaml:"ref" json:"ref"`
	ForeignKey []string `yaml:"foreign_key,omitempty" json:"foreign_key,omitempty"`
	On         *string  `yaml:"on,omitempty" json:"on,omitempty"`
}

// SourceNode is one source in a multi-source view.
type SourceNode struct {
	Name          string         `yaml:"name" json:"name"`
	From          string         `yaml:"from" json:"from"`
	PrimaryKey    []string       `yaml:"primary_key,omitempty" json:"primary_key,omitempty"`
	Relationships []Relationship `yaml:"relationships,omitempty" json:"relationships,omitempty"`
}

// SingleSourceMetricView is the v1.1 single-source metric view shape.
type SingleSourceMetricView struct {
	Version         string           `yaml:"version" json:"version"`
	Source          string           `yaml:"source" json:"source"`
	Joins           []Join           `yaml:"joins,omitempty" json:"joins,omitempty"`
	Filter          *string          `yaml:"filter,omitempty" json:"filter,omitempty"`
	Comment         *string          `yaml:"comment,omitempty" json:"comment,omitempty"`
	Parameters      []ParameterV11   `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Dimensions      []ColumnV11      `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	Measures        []ColumnV11      `yaml:"measures,omitempty" json:"measures,omitempty"`
	Materialization *Materialization `yaml:"materialization,omitempty" json:"materialization,omitempty"`
}

// UnmarshalYAML accepts "fields" as an alias for "dimensions".
func (v *SingleSourceMetricView) UnmarshalYAML(node *yaml.Node) error {
	// The local alias type strips this UnmarshalYAML method, so decoding the
	// inlined struct does not recurse. The sibling Fields field carries the
	// "fields" alias alongside the inlined "dimensions".
	type alias SingleSourceMetricView
	aux := struct {
		alias  `yaml:",inline"`
		Fields []ColumnV11 `yaml:"fields"`
	}{}
	if err := node.Decode(&aux); err != nil {
		return err
	}
	*v = SingleSourceMetricView(aux.alias)
	if len(v.Dimensions) == 0 {
		v.Dimensions = aux.Fields
	}
	return nil
}

// MultiSourceMetricView is the v1.1 multi-source metric view shape.
type MultiSourceMetricView struct {
	Version    string         `yaml:"version" json:"version"`
	ViewType   string         `yaml:"view_type" json:"view_type"`
	Sources    []SourceNode   `yaml:"sources" json:"sources"`
	Comment    *string        `yaml:"comment,omitempty" json:"comment,omitempty"`
	Parameters []ParameterV11 `yaml:"parameters,omitempty" json:"parameters,omitempty"`
	Dimensions []ColumnV11    `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	Measures   []ColumnV11    `yaml:"measures,omitempty" json:"measures,omitempty"`
}

// UnmarshalYAML defaults view_type to MULTI_SOURCE and accepts "fields" as an
// alias for "dimensions".
func (v *MultiSourceMetricView) UnmarshalYAML(node *yaml.Node) error {
	type alias MultiSourceMetricView
	aux := struct {
		alias  `yaml:",inline"`
		Fields []ColumnV11 `yaml:"fields"`
	}{}
	if err := node.Decode(&aux); err != nil {
		return err
	}
	*v = MultiSourceMetricView(aux.alias)
	if v.ViewType == "" {
		v.ViewType = viewTypeMultiSource
	}
	if len(v.Dimensions) == 0 {
		v.Dimensions = aux.Fields
	}
	return nil
}
