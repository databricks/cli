package metricviews

import (
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

// ValidateYAML checks input rules whose evidence is lost when YAML is decoded.
func ValidateYAML(node *yaml.Node) error {
	root := node
	if node.Kind == yaml.DocumentNode {
		if len(node.Content) == 0 {
			return errors.New("metricviews: missing YAML document")
		}
		root = node.Content[0]
	}
	version, viewType, hasSources := peek(root)
	switch version {
	case version01, version10:
		if err := validateYAMLColumns(root); err != nil {
			return err
		}
		return visitYAMLField(root, "materialization", validateYAMLMaterialization)
	default:
		if err := validateYAMLColumns(root); err != nil {
			return err
		}
		if err := visitYAMLSequence(root, "parameters", validateYAMLParameter); err != nil {
			return err
		}
		if viewType == viewTypeMultiSource || hasSources {
			return nil
		}
		return visitYAMLField(root, "materialization", validateYAMLMaterialization)
	}
}

// ParseAndValidate decodes a user-supplied definition after checking its shape.
func ParseAndValidate(data []byte) (*MetricView, error) {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("metricviews: parsing YAML: %w", err)
	}
	if err := ValidateYAML(&root); err != nil {
		return nil, err
	}
	m, err := decodeMetricView(&root)
	if err != nil {
		return nil, err
	}
	if err := Validate(m); err != nil {
		return nil, err
	}
	return m, nil
}

func validateYAMLColumns(node *yaml.Node) error {
	if err := rejectDimensionFieldConflict(node); err != nil {
		return err
	}
	field := "dimensions"
	if yamlField(node, "fields") != nil {
		field = "fields"
	}
	if err := visitYAMLSequence(node, field, validateYAMLColumn); err != nil {
		return err
	}
	return visitYAMLSequence(node, "measures", validateYAMLColumn)
}

func rejectDimensionFieldConflict(node *yaml.Node) error {
	if yamlField(node, "dimensions") != nil && yamlField(node, "fields") != nil {
		return errors.New("cannot specify both dimensions and fields")
	}
	return nil
}

func validateYAMLColumn(node *yaml.Node) error {
	return visitYAMLField(node, "format", validateYAMLFormat)
}

func validateYAMLMaterialization(node *yaml.Node) error {
	return visitYAMLSequence(node, "materialized_views", rejectDimensionFieldConflict)
}

func validateYAMLParameter(node *yaml.Node) error {
	seen := make(map[string]bool)
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i].Value
		switch key {
		case "name", "data_type", "default":
			if seen[key] {
				return fmt.Errorf("duplicate parameter field %q", key)
			}
			seen[key] = true
		default:
			return fmt.Errorf("unknown parameter field %q", key)
		}
	}
	return nil
}

func validateYAMLFormat(node *yaml.Node) error {
	typeNode := yamlField(node, "type")
	if typeNode == nil {
		return nil
	}
	typ := yamlScalar(typeNode)
	allowed, _ := formatAllowedFields(typ)
	if allowed == nil {
		return nil // The model validator reports unsupported types.
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		field := node.Content[i].Value
		if !allowed[field] {
			return fmt.Errorf("field %q is not valid for format type %q", field, typ)
		}
	}
	return nil
}

func visitYAMLField(node *yaml.Node, field string, visit func(*yaml.Node) error) error {
	value := yamlField(node, field)
	if value == nil {
		return nil
	}
	value = resolveYAMLAlias(value)
	if value.Tag == "!!null" {
		return nil
	}
	if err := visit(value); err != nil {
		return fmt.Errorf("%s: %w", field, err)
	}
	return nil
}

func visitYAMLSequence(node *yaml.Node, field string, visit func(*yaml.Node) error) error {
	return visitYAMLField(node, field, func(value *yaml.Node) error {
		if value.Kind != yaml.SequenceNode {
			return fmt.Errorf("expected a sequence, got kind %d", value.Kind)
		}
		for i, item := range value.Content {
			if err := visit(resolveYAMLAlias(item)); err != nil {
				return fmt.Errorf("item %d: %w", i, err)
			}
		}
		return nil
	})
}
