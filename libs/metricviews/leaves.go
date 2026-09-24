package metricviews

import "go.yaml.in/yaml/v3"

// WindowSpec is a window specification for a semi-additive measure.
type WindowSpec struct {
	Order        string `yaml:"order" json:"order"`
	Semiadditive string `yaml:"semiadditive" json:"semiadditive"`
	Range        string `yaml:"range" json:"range"`
	Offset       string `yaml:"offset,omitempty" json:"offset,omitempty"`
}

func (w *WindowSpec) UnmarshalYAML(node *yaml.Node) error {
	if err := requireYAMLFields(node, "order", "semiadditive", "range"); err != nil {
		return err
	}
	type alias WindowSpec
	return node.Decode((*alias)(w))
}

// RelyOptions captures reliability hints on a join.
type RelyOptions struct {
	AtMostOneMatch *bool `yaml:"at_most_one_match,omitempty" json:"at_most_one_match,omitempty"`
}

// Join is a join clause; joins nest recursively.
type Join struct {
	Name        string       `yaml:"name" json:"name"`
	Source      string       `yaml:"source" json:"source"`
	On          *string      `yaml:"on,omitempty" json:"on,omitempty"`
	Using       []string     `yaml:"using,omitempty" json:"using,omitempty"`
	Joins       []Join       `yaml:"joins,omitempty" json:"joins,omitempty"`
	Rely        *RelyOptions `yaml:"rely,omitempty" json:"rely,omitempty"`
	Cardinality *string      `yaml:"cardinality,omitempty" json:"cardinality,omitempty"`
}

func (j *Join) UnmarshalYAML(node *yaml.Node) error {
	if err := requireYAMLFields(node, "name", "source"); err != nil {
		return err
	}
	type alias Join
	return node.Decode((*alias)(j))
}

// MaterializedView is one entry in a Materialization.
type MaterializedView struct {
	Name       string   `yaml:"name" json:"name"`
	MVType     string   `yaml:"type" json:"type"`
	Dimensions []string `yaml:"dimensions,omitempty" json:"dimensions,omitempty"`
	Measures   []string `yaml:"measures,omitempty" json:"measures,omitempty"`
}

// UnmarshalYAML accepts "fields" as an alias for "dimensions".
func (m *MaterializedView) UnmarshalYAML(node *yaml.Node) error {
	if err := requireYAMLFields(node, "name", "type"); err != nil {
		return err
	}
	if err := rejectDimensionFieldConflict(node); err != nil {
		return err
	}
	type alias MaterializedView
	aux := struct {
		alias  `yaml:",inline"`
		Fields []string `yaml:"fields"`
	}{}
	if err := node.Decode(&aux); err != nil {
		return err
	}
	*m = MaterializedView(aux.alias)
	if len(m.Dimensions) == 0 {
		m.Dimensions = aux.Fields
	}
	return nil
}

// Materialization configures incremental materialization of a metric view.
type Materialization struct {
	Schedule          string             `yaml:"schedule" json:"schedule"`
	Mode              string             `yaml:"mode" json:"mode"`
	MaterializedViews []MaterializedView `yaml:"materialized_views" json:"materialized_views"`
}

func (m *Materialization) UnmarshalYAML(node *yaml.Node) error {
	if err := requireYAMLFields(node, "schedule", "mode", "materialized_views"); err != nil {
		return err
	}
	type alias Materialization
	return node.Decode((*alias)(m))
}
