package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle/internal/annotation"
)

// Version when bundle/schema/jsonschema.json was added to the repo.
var embeddedSchemaVersion = [3]int{0, 229, 0}

// updateSinceVersions updates annotation files with since_version for each field.
// It uses .last_processed_cli_version to track the last processed version and only
// processes new versions incrementally.
func updateSinceVersions(workDir string) error {
	annotationsPath := filepath.Join(workDir, "annotations.yml")
	overridesPath := filepath.Join(workDir, "annotations_openapi_overrides.yml")
	previousVersionFile := filepath.Join(workDir, ".last_processed_cli_version")

	// Load existing annotations
	annotations, err := annotation.Load(annotationsPath)
	if err != nil {
		return fmt.Errorf("loading annotations: %w", err)
	}

	overrides, err := annotation.Load(overridesPath)
	if err != nil {
		return fmt.Errorf("loading overrides: %w", err)
	}

	// Get all version tags
	allVersions, err := getVersionTags()
	if err != nil {
		return fmt.Errorf("getting version tags: %w", err)
	}
	if len(allVersions) == 0 {
		return errors.New("no version tags found")
	}

	// Read previous version to determine where to start
	previousVersion := readLastProcessedVersion(previousVersionFile)
	versions := filterVersionsAfter(allVersions, previousVersion)

	if len(versions) == 0 {
		fmt.Printf("Since versions are up to date (at %s)\n", previousVersion)
		return nil
	}

	fmt.Printf("Updating since_version annotations from %s to %s (%d versions)\n",
		versions[0], versions[len(versions)-1], len(versions))

	// Compute since versions for the new range
	sinceVersions, err := computeSinceVersions(versions)
	if err != nil {
		return fmt.Errorf("computing since versions: %w", err)
	}

	// Get current schema to filter out removed types
	currentSchema, err := getSchemaAtVersion(allVersions[len(allVersions)-1])
	if err != nil {
		return fmt.Errorf("getting current schema: %w", err)
	}
	currentFields := flattenSchema(currentSchema)
	sinceVersions = filterToCurrentFields(sinceVersions, currentFields)

	// Update annotations
	cliAdded := updateAnnotationsWithVersions(annotations, sinceVersions, overrides, true)
	sdkAdded := updateAnnotationsWithVersions(overrides, sinceVersions, annotations, false)

	// Save if there were changes
	if cliAdded > 0 {
		if err := annotations.Save(annotationsPath); err != nil {
			return fmt.Errorf("saving annotations: %w", err)
		}
		fmt.Printf("Added %d since_version entries to annotations.yml\n", cliAdded)
	}

	if sdkAdded > 0 {
		if err := overrides.Save(overridesPath); err != nil {
			return fmt.Errorf("saving overrides: %w", err)
		}
		fmt.Printf("Added %d since_version entries to annotations_openapi_overrides.yml\n", sdkAdded)
	}

	// Save the current version as the new previous version
	latestVersion := allVersions[len(allVersions)-1]
	if err := os.WriteFile(previousVersionFile, []byte(latestVersion+"\n"), 0o644); err != nil {
		return fmt.Errorf("writing previous version file: %w", err)
	}

	return nil
}

// readLastProcessedVersion reads the last processed version from the file.
func readLastProcessedVersion(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// filterVersionsAfter returns versions that come after the given version.
func filterVersionsAfter(versions []string, after string) []string {
	if after == "" {
		return versions
	}

	afterParsed, err := parseVersion(after)
	if err != nil {
		return versions
	}

	var result []string
	for _, v := range versions {
		parsed, err := parseVersion(v)
		if err != nil {
			continue
		}
		if compareVersions(parsed, afterParsed) > 0 {
			result = append(result, v)
		}
	}
	return result
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

		props, hasProps := valueMap["properties"].(map[string]any)
		_, hasOneOf := valueMap["oneOf"]
		_, hasAnyOf := valueMap["anyOf"]

		if hasProps || hasOneOf || hasAnyOf {
			if !hasProps {
				if oneOf, ok := valueMap["oneOf"].([]any); ok {
					for _, variant := range oneOf {
						if variantMap, ok := variant.(map[string]any); ok {
							if p, ok := variantMap["properties"].(map[string]any); ok {
								props = p
								break
							}
						}
					}
				}
				if props == nil {
					if anyOf, ok := valueMap["anyOf"].([]any); ok {
						for _, variant := range anyOf {
							if variantMap, ok := variant.(map[string]any); ok {
								if p, ok := variantMap["properties"].(map[string]any); ok {
									props = p
									break
								}
							}
						}
					}
				}
			}
			if props != nil {
				var propNames []string
				for propName := range props {
					propNames = append(propNames, propName)
				}
				result[currentPath] = propNames
			}
		} else if _, hasType := valueMap["type"]; hasType {
			continue
		} else {
			nested := walkDefs(valueMap, currentPath)
			maps.Copy(result, nested)
		}
	}
	return result
}

// flattenSchema extracts all field paths from a JSON schema.
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

// computeSinceVersions computes when each field was first introduced.
func computeSinceVersions(versions []string) (map[string]map[string]string, error) {
	allFields := make(map[string]bool)
	sinceVersions := make(map[string]string)

	for _, version := range versions {
		schema, err := getSchemaAtVersion(version)
		if err != nil {
			continue
		}

		currentFields := flattenSchema(schema)
		for field := range currentFields {
			if !allFields[field] {
				sinceVersions[field] = version
				allFields[field] = true
			}
		}
	}

	result := make(map[string]map[string]string)
	for fieldPath, version := range sinceVersions {
		lastDot := strings.LastIndex(fieldPath, ".")
		if lastDot == -1 {
			continue
		}
		typePath := fieldPath[:lastDot]
		propName := fieldPath[lastDot+1:]

		if result[typePath] == nil {
			result[typePath] = make(map[string]string)
		}
		result[typePath][propName] = version
	}

	return result, nil
}

// isBundleCliPath checks if a type path is from the CLI package (not SDK).
func isBundleCliPath(path string) bool {
	return strings.HasPrefix(path, "github.com/databricks/cli/")
}

// filterToCurrentFields removes fields that don't exist in currentFields.
func filterToCurrentFields(sinceVersions map[string]map[string]string, currentFields map[string]bool) map[string]map[string]string {
	result := make(map[string]map[string]string)
	for typePath, props := range sinceVersions {
		for propName, version := range props {
			fieldPath := typePath + "." + propName
			if currentFields[fieldPath] {
				if result[typePath] == nil {
					result[typePath] = make(map[string]string)
				}
				result[typePath][propName] = version
			}
		}
	}
	return result
}

// updateAnnotationsWithVersions updates annotations with since_version.
func updateAnnotationsWithVersions(
	annotations annotation.File,
	sinceVersions map[string]map[string]string,
	skipIfIn annotation.File,
	cliTypes bool,
) int {
	added := 0
	for typePath, props := range sinceVersions {
		isCli := isBundleCliPath(typePath)
		if cliTypes != isCli {
			continue
		}

		for propName, version := range props {
			if skipIfIn[typePath] != nil {
				if _, exists := skipIfIn[typePath][propName]; exists {
					continue
				}
			}

			if annotations[typePath] == nil {
				annotations[typePath] = make(map[string]annotation.Descriptor)
			}

			propData := annotations[typePath][propName]
			if propData.SinceVersion == "" {
				propData.SinceVersion = version
				annotations[typePath][propName] = propData
				added++
			}
		}
	}

	return added
}
