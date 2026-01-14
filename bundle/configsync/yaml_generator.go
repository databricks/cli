package configsync

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"gopkg.in/yaml.v3"
)

// resourceKeyToDynPath converts a resource key to a dyn.Path
// Example: "resources.jobs.my_job" -> Path{Key("resources"), Key("jobs"), Key("my_job")}
func resourceKeyToDynPath(resourceKey string) (dyn.Path, error) {
	if resourceKey == "" {
		return nil, errors.New("invalid resource key: empty string")
	}

	parts := strings.Split(resourceKey, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid resource key: %s", resourceKey)
	}

	path := make(dyn.Path, len(parts))
	for i, part := range parts {
		path[i] = dyn.Key(part)
	}

	return path, nil
}

// getResourceWithLocation retrieves a resource dyn.Value and its file location
// Uses the dynamic config value, not typed structures
func getResourceWithLocation(configValue dyn.Value, resourceKey string) (dyn.Value, dyn.Location, error) {
	path, err := resourceKeyToDynPath(resourceKey)
	if err != nil {
		return dyn.NilValue, dyn.Location{}, err
	}

	resource, err := dyn.GetByPath(configValue, path)
	if err != nil {
		return dyn.NilValue, dyn.Location{}, fmt.Errorf("resource %s not found: %w", resourceKey, err)
	}

	return resource, resource.Location(), nil
}

// structpathToDynPath converts a structpath string to a dyn.Path
// Example: "tasks[0].timeout_seconds" -> Path{Key("tasks"), Index(0), Key("timeout_seconds")}
// Also supports "tasks[task_key='my_task']" syntax for array element selection by field value
func structpathToDynPath(_ context.Context, pathStr string, baseValue dyn.Value) (dyn.Path, error) {
	node, err := structpath.Parse(pathStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	nodes := node.AsSlice()

	var dynPath dyn.Path
	currentValue := baseValue

	for _, n := range nodes {
		// Check for string key (field access)
		if key, ok := n.StringKey(); ok {
			dynPath = append(dynPath, dyn.Key(key))

			// Update currentValue for next iteration
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Key(key)})
			}
			continue
		}

		// Check for numeric index
		if idx, ok := n.Index(); ok {
			dynPath = append(dynPath, dyn.Index(idx))

			// Update currentValue for next iteration
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Index(idx)})
			}
			continue
		}

		// Check for key-value selector: [key='value']
		if key, value, ok := n.KeyValue(); ok {
			// Need to search the array to find the matching index
			if !currentValue.IsValid() || currentValue.Kind() != dyn.KindSequence {
				return nil, fmt.Errorf("cannot apply [key='value'] selector to non-array value at path %s", dynPath.String())
			}

			seq, _ := currentValue.AsSequence()
			foundIndex := -1

			for i, elem := range seq {
				keyValue, err := dyn.GetByPath(elem, dyn.Path{dyn.Key(key)})
				if err != nil {
					continue
				}

				// Compare the key value
				if keyValue.Kind() == dyn.KindString && keyValue.MustString() == value {
					foundIndex = i
					break
				}
			}

			if foundIndex == -1 {
				return nil, fmt.Errorf("no array element found with %s='%s' at path %s", key, value, dynPath.String())
			}

			dynPath = append(dynPath, dyn.Index(foundIndex))
			currentValue = seq[foundIndex]
			continue
		}

		// Skip wildcards or other special node types
		if n.DotStar() || n.BracketStar() {
			return nil, errors.New("wildcard patterns are not supported in field paths")
		}
	}

	return dynPath, nil
}

// applyChanges applies all field changes to a resource dyn.Value
func applyChanges(ctx context.Context, resource dyn.Value, changes deployplan.Changes) (dyn.Value, error) {
	result := resource

	for fieldPath, changeDesc := range changes {
		// Convert structpath to dyn.Path
		dynPath, err := structpathToDynPath(ctx, fieldPath, result)
		if err != nil {
			log.Warnf(ctx, "Failed to parse field path %s: %v", fieldPath, err)
			continue
		}

		// Set the remote value at the path
		remoteValue := dyn.V(changeDesc.Remote)
		result, err = dyn.SetByPath(result, dynPath, remoteValue)
		if err != nil {
			log.Warnf(ctx, "Failed to set value at path %s: %v", fieldPath, err)
			continue
		}
	}

	return result, nil
}

// dynValueToYAML converts a dyn.Value to a YAML string
func dynValueToYAML(v dyn.Value) (string, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(v.AsAny()); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// parseResourceKey extracts resource type and name from a resource key
// Example: "resources.jobs.my_job" -> type="jobs", name="my_job"
func parseResourceKey(resourceKey string) (resourceType, resourceName string, err error) {
	parts := strings.Split(resourceKey, ".")
	if len(parts) < 3 || parts[0] != "resources" {
		return "", "", fmt.Errorf("invalid resource key format: %s (expected resources.TYPE.NAME)", resourceKey)
	}

	return parts[1], parts[2], nil
}

// findResourceInFile searches for a resource within a loaded file's dyn.Value
func findResourceInFile(_ context.Context, fileValue dyn.Value, resourceType, resourceName, targetName string) (dyn.Value, dyn.Path, error) {
	patternsToCheck := []dyn.Path{
		{dyn.Key("targets"), dyn.Key(targetName), dyn.Key("resources"), dyn.Key(resourceType), dyn.Key(resourceName)},
		{dyn.Key("resources"), dyn.Key(resourceType), dyn.Key(resourceName)},
	}

	for _, pattern := range patternsToCheck {
		resource, err := dyn.GetByPath(fileValue, pattern)
		if err == nil {
			return resource, pattern, nil
		}
	}

	directPath := dyn.Path{dyn.Key("resources"), dyn.Key(resourceType), dyn.Key(resourceName)}
	resource, err := dyn.GetByPath(fileValue, directPath)
	if err == nil {
		return resource, directPath, nil
	}

	return dyn.NilValue, nil, fmt.Errorf("resource %s.%s not found in file", resourceType, resourceName)
}

// GenerateYAMLFiles generates YAML files for the given changes.
func GenerateYAMLFiles(ctx context.Context, b *bundle.Bundle, changes map[string]deployplan.Changes) ([]FileChange, error) {
	configValue := b.Config.Value()

	fileChanges := make(map[string][]struct {
		resourceKey string
		changes     deployplan.Changes
	})

	for resourceKey, resourceChanges := range changes {
		_, loc, err := getResourceWithLocation(configValue, resourceKey)
		if err != nil {
			log.Warnf(ctx, "Failed to find resource %s in bundle config: %v", resourceKey, err)
			continue
		}

		filePath := loc.File
		fileChanges[filePath] = append(fileChanges[filePath], struct {
			resourceKey string
			changes     deployplan.Changes
		}{resourceKey, resourceChanges})
	}

	var result []FileChange

	for filePath, resourcesInFile := range fileChanges {
		content, err := os.ReadFile(filePath)
		if err != nil {
			log.Warnf(ctx, "Failed to read file %s: %v", filePath, err)
			continue
		}

		// Load file as dyn.Value
		fileValue, err := yamlloader.LoadYAML(filePath, bytes.NewBuffer(content))
		if err != nil {
			log.Warnf(ctx, "Failed to parse YAML file %s: %v", filePath, err)
			continue
		}

		// Apply changes for each resource in this file
		for _, item := range resourcesInFile {
			// Parse resource key
			resourceType, resourceName, err := parseResourceKey(item.resourceKey)
			if err != nil {
				log.Warnf(ctx, "Failed to parse resource key %s: %v", item.resourceKey, err)
				continue
			}

			// Find resource in loaded file
			resource, resourcePath, err := findResourceInFile(ctx, fileValue, resourceType, resourceName, b.Config.Bundle.Target)
			if err != nil {
				log.Warnf(ctx, "Failed to find resource %s in file %s: %v", item.resourceKey, filePath, err)
				continue
			}

			// Apply changes to the resource
			modifiedResource, err := applyChanges(ctx, resource, item.changes)
			if err != nil {
				log.Warnf(ctx, "Failed to apply changes to resource %s: %v", item.resourceKey, err)
				continue
			}

			// Update the file's dyn.Value with modified resource
			fileValue, err = dyn.SetByPath(fileValue, resourcePath, modifiedResource)
			if err != nil {
				log.Warnf(ctx, "Failed to update file value for resource %s: %v", item.resourceKey, err)
				continue
			}
		}

		// Convert modified dyn.Value to YAML string
		modifiedContent, err := dynValueToYAML(fileValue)
		if err != nil {
			log.Warnf(ctx, "Failed to convert modified value to YAML for file %s: %v", filePath, err)
			continue
		}

		result = append(result, FileChange{
			Path:            filePath,
			OriginalContent: string(content),
			ModifiedContent: modifiedContent,
		})
	}

	return result, nil
}
