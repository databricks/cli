package yamlutil

import (
	"bufio"
	"errors"
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

// YamlNodeToDynValue converts a yaml.Node to dyn.Value, converting camelCase field names to snake_case.
func YamlNodeToDynValue(node *yaml.Node) (dyn.Value, error) {
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
			value, err := YamlNodeToDynValue(valueNode)
			if err != nil {
				return dyn.NilValue, err
			}

			pairs = append(pairs, dyn.Pair{Key: key, Value: value})
		}
		return dyn.NewValue(dyn.NewMappingFromPairs(pairs), []dyn.Location{}), nil

	case yaml.SequenceNode:
		items := make([]dyn.Value, 0, len(node.Content))
		for _, itemNode := range node.Content {
			item, err := YamlNodeToDynValue(itemNode)
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
		return YamlNodeToDynValue(node.Alias)

	default:
		return dyn.NilValue, fmt.Errorf("unsupported YAML node kind: %v", node.Kind)
	}
}

// AddBlankLinesBetweenTopLevelKeys adds blank lines between top-level sections in YAML.
func AddBlankLinesBetweenTopLevelKeys(filename string) error {
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

// InlineAppConfigFile reads app.yml or app.yaml, inlines it into the app value, and returns the filename.
func InlineAppConfigFile(appValue *dyn.Value) (string, error) {
	var appConfigFile string
	var appConfigData []byte
	var err error

	for _, filename := range []string{"app.yml", "app.yaml"} {
		if _, statErr := os.Stat(filename); statErr == nil {
			appConfigFile = filename
			appConfigData, err = os.ReadFile(filename)
			if err != nil {
				return "", fmt.Errorf("failed to read %s: %w", filename, err)
			}
			break
		}
	}

	if appConfigFile == "" {
		return "", nil
	}

	var appConfigNode yaml.Node
	err = yaml.Unmarshal(appConfigData, &appConfigNode)
	if err != nil {
		return "", fmt.Errorf("failed to parse %s: %w", appConfigFile, err)
	}

	appConfigValue, err := YamlNodeToDynValue(&appConfigNode)
	if err != nil {
		return "", fmt.Errorf("failed to convert app config: %w", err)
	}

	appConfigMap, ok := appConfigValue.AsMap()
	if !ok {
		return "", errors.New("app config is not a map")
	}

	appMap, ok := appValue.AsMap()
	if !ok {
		return "", errors.New("app value is not a map")
	}

	newPairs := make([]dyn.Pair, 0, len(appMap.Pairs())+2)
	newPairs = append(newPairs, appMap.Pairs()...)

	var configPairs []dyn.Pair
	var resourcesValue dyn.Value

	for _, pair := range appConfigMap.Pairs() {
		key := pair.Key.MustString()
		switch key {
		case "command", "env":
			configPairs = append(configPairs, pair)
		case "resources":
			resourcesValue = pair.Value
		}
	}

	if len(configPairs) > 0 {
		newPairs = append(newPairs, dyn.Pair{
			Key:   dyn.V("config"),
			Value: dyn.NewValue(dyn.NewMappingFromPairs(configPairs), []dyn.Location{}),
		})
	}

	if resourcesValue.Kind() != dyn.KindInvalid {
		newPairs = append(newPairs, dyn.Pair{
			Key:   dyn.V("resources"),
			Value: resourcesValue,
		})
	}

	newMapping := dyn.NewMappingFromPairs(newPairs)
	*appValue = dyn.NewValue(newMapping, appValue.Locations())

	return appConfigFile, nil
}
