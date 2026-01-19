package configsync

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/palantir/pkg/yamlpatch/gopkgv3yamlpatcher"
	"github.com/palantir/pkg/yamlpatch/yamlpatch"
)

type resolvedChanges map[string]*deployplan.ChangeDesc

// ApplyChangesToYAML generates YAML files for the given changes.
func ApplyChangesToYAML(ctx context.Context, b *bundle.Bundle, planChanges map[string]deployplan.Changes) ([]FileChange, error) {
	changesByFile, err := getResolvedFieldChanges(ctx, b, planChanges)
	if err != nil {
		return nil, err
	}

	var result []FileChange
	targetName := b.Config.Bundle.Target

	for filePath, changes := range changesByFile {
		originalContent, err := os.ReadFile(filePath)
		if err != nil {
			log.Warnf(ctx, "Failed to read file %s: %v", filePath, err)
			continue
		}

		modifiedContent, err := applyChanges(ctx, filePath, changes, targetName)
		if err != nil {
			log.Warnf(ctx, "Failed to apply changes to file %s: %v", filePath, err)
			continue
		}

		if modifiedContent != string(originalContent) {
			result = append(result, FileChange{
				Path:            filePath,
				OriginalContent: string(originalContent),
				ModifiedContent: modifiedContent,
			})
		}
	}

	return result, nil
}

// applyChanges applies all field changes to a YAML
func applyChanges(ctx context.Context, filePath string, changes resolvedChanges, targetName string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	for fieldPath, changeDesc := range changes {
		jsonPointer := strPathToJSONPointer(fieldPath)

		jsonPointers := []string{jsonPointer}
		if targetName != "" {
			targetPrefix := "/targets/" + targetName
			jsonPointers = append(jsonPointers, targetPrefix+jsonPointer)
		}

		hasConfigValue := changeDesc.Old != nil || changeDesc.New != nil
		isRemoval := changeDesc.Remote == nil && hasConfigValue
		isReplacement := changeDesc.Remote != nil && hasConfigValue
		isAddition := changeDesc.Remote != nil && !hasConfigValue

		for _, jsonPointer := range jsonPointers {
			path, err := yamlpatch.ParsePath(jsonPointer)
			if err != nil {
				return "", fmt.Errorf("failed to parse JSON Pointer %s: %w", jsonPointer, err)
			}

			var testOp yamlpatch.Operation
			if isRemoval {
				testOp = yamlpatch.Operation{
					Type: yamlpatch.OperationRemove,
					Path: path,
				}
			} else if isReplacement {
				testOp = yamlpatch.Operation{
					Type:  yamlpatch.OperationReplace,
					Path:  path,
					Value: changeDesc.Remote,
				}
			} else if isAddition {
				testOp = yamlpatch.Operation{
					Type:  yamlpatch.OperationAdd,
					Path:  path,
					Value: changeDesc.Remote,
				}
			} else {
				log.Warnf(ctx, "Unknown operation type for field %s", fieldPath)
				continue
			}

			patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))
			modifiedContent, err := patcher.Apply(content, yamlpatch.Patch{testOp})
			if err == nil {
				content = modifiedContent
				log.Debugf(ctx, "Applied %s change to %s", testOp.Type, jsonPointer)
				break
			} else {
				log.Debugf(ctx, "Failed to apply change to %s: %v", jsonPointer, err)
			}
		}
	}

	return string(content), nil
}

// getResolvedFieldChanges builds a map from file paths to lists of field changes
func getResolvedFieldChanges(ctx context.Context, b *bundle.Bundle, planChanges map[string]deployplan.Changes) (map[string]resolvedChanges, error) {
	resolvedChangesByFile := make(map[string]resolvedChanges)

	for resourceKey, resourceChanges := range planChanges {
		for fieldPath, changeDesc := range resourceChanges {
			fullPath := resourceKey + "." + fieldPath

			resolvedPath, err := resolveSelectors(fullPath, b)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve selectors in path %s: %w", fullPath, err)
			}

			loc := b.Config.GetLocation(resolvedPath)
			filePath := loc.File

			// If field has no location, find the parent resource's location to then add a new field
			if filePath == "" {
				filePath = findResourceFileLocation(ctx, b, resourceKey)
				if filePath == "" {
					continue
				}
				log.Debugf(ctx, "Field %s has no location, using resource location: %s", fullPath, filePath)
			}

			if _, ok := resolvedChangesByFile[filePath]; !ok {
				resolvedChangesByFile[filePath] = make(resolvedChanges)
			}
			resolvedChangesByFile[filePath][resolvedPath] = changeDesc
		}
	}

	return resolvedChangesByFile, nil
}

// strPathToJSONPointer converts a structpath string to JSON Pointer format.
// Example: "resources.jobs.test[0].name" -> "/resources/jobs/test/0/name"
func strPathToJSONPointer(pathStr string) string {
	if pathStr == "" {
		return ""
	}
	res := strings.ReplaceAll(pathStr, ".", "/")
	res = strings.ReplaceAll(res, "[", "/")
	res = strings.ReplaceAll(res, "]", "")
	return "/" + res
}

// findResourceFileLocation finds the file where a resource is defined.
// It checks both the root resources and target-specific overrides,
// preferring the target override if it exists.
func findResourceFileLocation(_ context.Context, b *bundle.Bundle, resourceKey string) string {
	targetName := b.Config.Bundle.Target

	if targetName != "" {
		targetPath := "targets." + targetName + "." + resourceKey
		loc := b.Config.GetLocation(targetPath)
		if loc.File != "" {
			return loc.File
		}
	}

	loc := b.Config.GetLocation(resourceKey)
	return loc.File
}
