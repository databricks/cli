package apps

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"go.yaml.in/yaml/v3"
)

// camelToSnake converts a camelCase string to snake_case.
// Examples: valueFrom -> value_from, myValue -> my_value, ID -> id
func camelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if the previous character was lowercase or if the next character is lowercase
			// This handles cases like "ID" -> "id" vs "myID" -> "my_id"
			prevLower := i > 0 && s[i-1] >= 'a' && s[i-1] <= 'z'
			nextLower := i+1 < len(s) && s[i+1] >= 'a' && s[i+1] <= 'z'
			if prevLower || nextLower {
				result.WriteByte('_')
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// yamlNodeToDynValue converts a yaml.Node to dyn.Value, converting camelCase field names to snake_case.
func yamlNodeToDynValue(node *yaml.Node) (dyn.Value, error) {
	// yaml.Unmarshal wraps the document in a Document node, get the actual content
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		node = node.Content[0]
	}

	switch node.Kind {
	case yaml.MappingNode:
		pairs := make([]dyn.Pair, 0, len(node.Content)/2)
		for i := 0; i < len(node.Content); i += 2 {
			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Convert camelCase keys to snake_case
			snakeKey := camelToSnake(keyNode.Value)
			key := dyn.V(snakeKey)
			value, err := yamlNodeToDynValue(valueNode)
			if err != nil {
				return dyn.NilValue, err
			}

			pairs = append(pairs, dyn.Pair{Key: key, Value: value})
		}
		return dyn.NewValue(dyn.NewMappingFromPairs(pairs), []dyn.Location{}), nil

	case yaml.SequenceNode:
		items := make([]dyn.Value, 0, len(node.Content))
		for _, itemNode := range node.Content {
			item, err := yamlNodeToDynValue(itemNode)
			if err != nil {
				return dyn.NilValue, err
			}
			items = append(items, item)
		}
		return dyn.V(items), nil

	case yaml.ScalarNode:
		// Try to parse as different types
		switch node.Tag {
		case "!!str", "":
			return dyn.V(node.Value), nil
		case "!!bool":
			if node.Value == "true" {
				return dyn.V(true), nil
			}
			return dyn.V(false), nil
		case "!!int":
			var i int64
			if err := yaml.Unmarshal([]byte(node.Value), &i); err != nil {
				return dyn.NilValue, err
			}
			return dyn.V(i), nil
		case "!!float":
			var f float64
			if err := yaml.Unmarshal([]byte(node.Value), &f); err != nil {
				return dyn.NilValue, err
			}
			return dyn.V(f), nil
		case "!!null":
			return dyn.NilValue, nil
		default:
			// Default to string
			return dyn.V(node.Value), nil
		}

	case yaml.AliasNode:
		return yamlNodeToDynValue(node.Alias)

	default:
		return dyn.NilValue, fmt.Errorf("unsupported YAML node kind: %v", node.Kind)
	}
}

// addBlankLinesBetweenTopLevelKeys adds blank lines between top-level sections in YAML.
func addBlankLinesBetweenTopLevelKeys(filename string) error {
	// Read the file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	// Add blank lines before top-level keys (lines that don't start with space/tab and contain ':')
	var result []string
	for i, line := range lines {
		// Add blank line before top-level keys (except the first line)
		if i > 0 && len(line) > 0 && line[0] != ' ' && line[0] != '\t' && strings.Contains(line, ":") {
			result = append(result, "")
		}
		result = append(result, line)
	}

	// Write back to file
	file, err = os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range result {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}
