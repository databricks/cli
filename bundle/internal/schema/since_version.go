package main

import (
	"encoding/json"
	"fmt"
	"maps"
	"os/exec"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/jsonschema"
)

// Version when bundle/schema/jsonschema.json was added to the repo.
var embeddedSchemaVersion = [3]int{0, 229, 0}

// computeSinceVersions computes when each field was first introduced by analyzing git history.
// It returns a map from "typePath.fieldName" to the version string (e.g., "v0.229.0").
// This function always recomputes versions at runtime without storing state.
func computeSinceVersions() (map[string]string, error) {
	versions, err := getVersionTags()
	if err != nil {
		return nil, fmt.Errorf("getting version tags: %w", err)
	}
	if len(versions) == 0 {
		return nil, nil
	}

	sinceVersions := make(map[string]string)
	for _, version := range versions {
		schema, err := getSchemaAtVersion(version)
		if err != nil {
			continue
		}

		for field := range flattenSchema(schema) {
			if _, seen := sinceVersions[field]; !seen {
				sinceVersions[field] = version
			}
		}
	}

	return sinceVersions, nil
}

// parseVersion parses a version tag like "v0.228.0" into [0, 228, 0].
func parseVersion(tag string) ([3]int, error) {
	tag = strings.TrimPrefix(tag, "v")
	parts := strings.Split(tag, ".")
	if len(parts) < 3 {
		return [3]int{}, fmt.Errorf("invalid version tag: %s", tag)
	}
	var v [3]int
	for i := range 3 {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return [3]int{}, fmt.Errorf("invalid version component: %s", parts[i])
		}
		v[i] = n
	}
	return v, nil
}

// compareVersions returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersions(a, b [3]int) int {
	for i := range 3 {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return 0
}

// getVersionTags returns sorted list of version tags from git (oldest first).
func getVersionTags() ([]string, error) {
	cmd := exec.Command("git", "tag", "--list", "v*", "--sort=version:refname")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get git tags: %w", err)
	}

	var tags []string
	for _, line := range strings.Split(string(output), "\n") {
		tag := strings.TrimSpace(line)
		if tag == "" {
			continue
		}
		v, err := parseVersion(tag)
		if err != nil {
			continue
		}
		if compareVersions(v, embeddedSchemaVersion) >= 0 {
			tags = append(tags, tag)
		}
	}
	return tags, nil
}

// getSchemaAtVersion extracts the JSON schema from the embedded file at a given version.
func getSchemaAtVersion(version string) (map[string]any, error) {
	cmd := exec.Command("git", "show", version+":bundle/schema/jsonschema.json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get schema at %s: %w", version, err)
	}

	var schema map[string]any
	if err := json.Unmarshal(output, &schema); err != nil {
		return nil, fmt.Errorf("failed to parse schema at %s: %w", version, err)
	}
	return schema, nil
}

// flattenSchema extracts all field paths from a JSON schema.
// Returns a map of "typePath.fieldName" -> true for all fields in the schema.
func flattenSchema(schema map[string]any) map[string]bool {
	fields := make(map[string]bool)

	if defs, ok := schema["$defs"].(map[string]any); ok {
		typeDefs := walkDefs(defs, "")
		for typePath, propNames := range typeDefs {
			for _, propName := range propNames {
				fields[typePath+"."+propName] = true
			}
		}
	}

	rootType := "github.com/databricks/cli/bundle/config.Root"
	if props, ok := schema["properties"].(map[string]any); ok {
		for propName := range props {
			fields[rootType+"."+propName] = true
		}
	}

	return fields
}

// walkDefs recursively walks $defs to extract type definitions.
func walkDefs(defs map[string]any, prefix string) map[string][]string {
	result := make(map[string][]string)

	for key, value := range defs {
		valueMap, ok := value.(map[string]any)
		if !ok {
			continue
		}

		currentPath := prefix
		if currentPath != "" {
			currentPath += "/" + key
		} else {
			currentPath = key
		}

		props := extractProperties(valueMap)
		if props != nil {
			var propNames []string
			for propName := range props {
				propNames = append(propNames, propName)
			}
			result[currentPath] = propNames
		} else if _, hasType := valueMap["type"]; !hasType {
			// It's a nested namespace, recurse into it
			maps.Copy(result, walkDefs(valueMap, currentPath))
		}
	}
	return result
}

// extractProperties extracts the properties map from a schema definition,
// checking direct properties first, then oneOf/anyOf variants.
func extractProperties(valueMap map[string]any) map[string]any {
	if props, ok := valueMap["properties"].(map[string]any); ok {
		return props
	}

	// Check oneOf and anyOf variants for properties
	for _, key := range []string{"oneOf", "anyOf"} {
		if variants, ok := valueMap[key].([]any); ok {
			for _, variant := range variants {
				if variantMap, ok := variant.(map[string]any); ok {
					if props, ok := variantMap["properties"].(map[string]any); ok {
						return props
					}
				}
			}
		}
	}

	return nil
}

// addSinceVersionToSchema applies sinceVersion annotations to the generated schema.
// The sinceVersions map uses keys in the format "typePath.fieldName".
func addSinceVersionToSchema(s *jsonschema.Schema, sinceVersions map[string]string) {
	if sinceVersions == nil || s == nil {
		return
	}

	// Apply to root properties
	rootType := "github.com/databricks/cli/bundle/config.Root"
	for propName, prop := range s.Properties {
		key := rootType + "." + propName
		if version, ok := sinceVersions[key]; ok {
			prop.SinceVersion = version
		}
	}

	// Apply to $defs - the definitions are nested maps like:
	// {"github.com": {"databricks": {"cli": {"bundle": {"config.Root": Schema{...}}}}}}
	if s.Definitions == nil {
		return
	}

	walkDefinitions(s.Definitions, "", sinceVersions)
}

// walkDefinitions recursively walks the nested definitions map and applies sinceVersion.
func walkDefinitions(defs map[string]any, pathPrefix string, sinceVersions map[string]string) {
	for key, value := range defs {
		var currentPath string
		if pathPrefix != "" {
			currentPath = pathPrefix + "/" + key
		} else {
			currentPath = key
		}

		// Try to convert to *jsonschema.Schema
		if schema, ok := value.(*jsonschema.Schema); ok {
			addSinceVersionToProperties(schema, currentPath, sinceVersions)
			continue
		}

		// Try to convert to jsonschema.Schema (non-pointer)
		if schema, ok := value.(jsonschema.Schema); ok {
			addSinceVersionToProperties(&schema, currentPath, sinceVersions)
			// Update the map with the modified schema
			defs[key] = schema
			continue
		}

		// Otherwise, it's a nested map - recurse into it
		if nestedMap, ok := value.(map[string]any); ok {
			walkDefinitions(nestedMap, currentPath, sinceVersions)
		}
	}
}

// addSinceVersionToProperties applies sinceVersion to schema properties.
func addSinceVersionToProperties(s *jsonschema.Schema, typePath string, sinceVersions map[string]string) {
	if s == nil {
		return
	}

	// Apply to direct properties
	for propName, prop := range s.Properties {
		key := typePath + "." + propName
		if version, ok := sinceVersions[key]; ok {
			prop.SinceVersion = version
		}
	}

	// Handle OneOf variants (need to use index to modify in place)
	for i := range s.OneOf {
		addSinceVersionToProperties(&s.OneOf[i], typePath, sinceVersions)
	}

	// Handle AnyOf variants (need to use index to modify in place)
	for i := range s.AnyOf {
		addSinceVersionToProperties(&s.AnyOf[i], typePath, sinceVersions)
	}
}
