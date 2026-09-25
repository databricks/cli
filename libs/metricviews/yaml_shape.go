package metricviews

import (
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

func yamlField(node *yaml.Node, field string) *yaml.Node {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == field {
			return node.Content[i+1]
		}
	}
	return nil
}

func resolveYAMLAlias(node *yaml.Node) *yaml.Node {
	for node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}

func yamlScalar(node *yaml.Node) string {
	return resolveYAMLAlias(node).Value
}

func requireYAMLFields(node *yaml.Node, fields ...string) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("expected a mapping, got kind %d", node.Kind)
	}
	for _, field := range fields {
		value := yamlField(node, field)
		if value == nil {
			return fmt.Errorf("missing required field %q", field)
		}
		value = resolveYAMLAlias(value)
		if value.Tag == "!!null" {
			return fmt.Errorf("required field %q cannot be null", field)
		}
	}
	return nil
}

func rejectDimensionFieldConflict(node *yaml.Node) error {
	if yamlField(node, "dimensions") != nil && yamlField(node, "fields") != nil {
		return errors.New("cannot specify both dimensions and fields")
	}
	return nil
}
