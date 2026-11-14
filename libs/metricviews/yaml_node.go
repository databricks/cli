package metricviews

import "go.yaml.in/yaml/v3"

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
