package configsync

import (
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/palantir/pkg/yamlpatch/gopkgv3yamlpatcher"
	"github.com/palantir/pkg/yamlpatch/yamlpatch"
)

// applyChanges applies all field changes to a YAML
func applyChanges(ctx context.Context, filePath string, fieldLocations fieldLocations, targetName string) (string, error) {
	// Load file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// Build yamlpatch operations
	var operations yamlpatch.Patch
	for jsonPointer, changeDesc := range fieldLocations {
		// Use the remote value directly - yamlpatch handles serialization
		yamlValue := changeDesc.Remote

		jsonPointers := []string{jsonPointer}
		if targetName != "" {
			targetPrefix := "/targets/" + targetName
			jsonPointers = append(jsonPointers, targetPrefix+jsonPointer)
		}

		var successfulPath string
		for _, jsonPointer := range jsonPointers {
			path, err := yamlpatch.ParsePath(jsonPointer)
			if err != nil {
				continue
			}

			testOp := yamlpatch.Operation{
				Type:  yamlpatch.OperationReplace,
				Path:  path,
				Value: yamlValue,
			}

			patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))
			_, err = patcher.Apply(content, yamlpatch.Patch{testOp})
			if err == nil {
				successfulPath = jsonPointer
				break
			}
		}

		if successfulPath == "" {
			log.Warnf(ctx, "Failed to find valid path for %s", jsonPointers)
			continue
		}

		// Parse JSON Pointer path
		path, err := yamlpatch.ParsePath(successfulPath)
		if err != nil {
			log.Warnf(ctx, "Failed to parse JSON Pointer %s: %v", successfulPath, err)
			continue
		}

		// Create Replace operation
		op := yamlpatch.Operation{
			Type:  yamlpatch.OperationReplace,
			Path:  path,
			Value: yamlValue,
		}
		operations = append(operations, op)
	}

	// Create patcher and apply all patches
	patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))
	modifiedContent, err := patcher.Apply(content, operations)
	if err != nil {
		return "", fmt.Errorf("failed to apply patches to %s: %w", filePath, err)
	}

	return string(modifiedContent), nil
}

type fieldLocations map[string]*deployplan.ChangeDesc

// getFieldLocations builds a map from file paths to lists of field changes
func getFieldLocations(ctx context.Context, b *bundle.Bundle, changes map[string]deployplan.Changes) (map[string]fieldLocations, error) {
	configValue := b.Config.Value()
	targetName := b.Config.Bundle.Target
	locationsByFile := make(map[string]fieldLocations)

	for resourceKey, resourceChanges := range changes {
		for fieldPath, changeDesc := range resourceChanges {
			fullPath := resourceKey + "." + fieldPath

			var found bool
			var filePath string
			var resolvedPath dyn.Path

			dynPath, err := structpathToDynPath(ctx, fullPath, configValue)
			if err != nil {
				log.Warnf(ctx, "Failed to convert path %s to dyn.Path: %v", fullPath, err)
				continue
			}

			dynPathWithTarget := append(dyn.Path{dyn.Key("targets"), dyn.Key(targetName)}, dynPath...)
			paths := []dyn.Path{dynPathWithTarget, dynPath}

			for _, path := range paths {
				value, err := dyn.GetByPath(configValue, path)
				if err != nil {
					log.Debugf(ctx, "Path %s not found in config: %v", path.String(), err)
					continue
				}

				filePath = value.Location().File
				resolvedPath = path
				found = true
				break
			}

			if !found {
				log.Warnf(ctx, "Failed to find location for %s", fullPath)
				continue
			}

			jsonPointer := dynPathToJSONPointer(resolvedPath)

			if _, ok := locationsByFile[filePath]; !ok {
				locationsByFile[filePath] = make(fieldLocations)
			}
			locationsByFile[filePath][jsonPointer] = changeDesc
		}
	}

	return locationsByFile, nil
}

// ApplyChangesToYAML generates YAML files for the given changes.
func ApplyChangesToYAML(ctx context.Context, b *bundle.Bundle, changes map[string]deployplan.Changes) ([]FileChange, error) {
	locationsByFile, err := getFieldLocations(ctx, b, changes)
	if err != nil {
		return nil, err
	}

	var result []FileChange
	targetName := b.Config.Bundle.Target

	for filePath, jsonPointers := range locationsByFile {
		originalContent, err := os.ReadFile(filePath)
		if err != nil {
			log.Warnf(ctx, "Failed to read file %s: %v", filePath, err)
			continue
		}

		modifiedContent, err := applyChanges(ctx, filePath, jsonPointers, targetName)
		if err != nil {
			log.Warnf(ctx, "Failed to apply changes to file %s: %v", filePath, err)
			continue
		}

		result = append(result, FileChange{
			Path:            filePath,
			OriginalContent: string(originalContent),
			ModifiedContent: modifiedContent,
		})
	}

	return result, nil
}
