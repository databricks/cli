package configsync

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/palantir/pkg/yamlpatch/gopkgv3yamlpatcher"
	"github.com/palantir/pkg/yamlpatch/yamlpatch"
)

type resolvedChanges map[string]*ConfigChangeDesc

// ApplyChangesToYAML generates YAML files for the given changes.
func ApplyChangesToYAML(ctx context.Context, b *bundle.Bundle, configChanges map[string]map[string]*ConfigChangeDesc) ([]FileChange, error) {
	changesByFile, err := getResolvedFieldChanges(ctx, b, configChanges)
	if err != nil {
		return nil, err
	}

	var result []FileChange
	targetName := b.Config.Bundle.Target

	for filePath, changes := range changesByFile {
		originalContent, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		modifiedContent, err := applyChanges(ctx, filePath, changes, targetName)
		if err != nil {
			return nil, fmt.Errorf("failed to apply changes to file %s: %w", filePath, err)
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

type parentNode struct {
	path        yamlpatch.Path
	missingPath yamlpatch.Path
}

// applyChanges applies all field changes to a YAML
func applyChanges(ctx context.Context, filePath string, changes resolvedChanges, targetName string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	fieldPaths := make([]string, 0, len(changes))
	for fieldPath := range changes {
		fieldPaths = append(fieldPaths, fieldPath)
	}
	sort.Strings(fieldPaths)

	for _, fieldPath := range fieldPaths {
		configChange := changes[fieldPath]
		fieldJsonPointer, err := strPathToJSONPointer(fieldPath)
		if err != nil {
			return "", fmt.Errorf("failed to convert field path %q to JSON pointer: %w", fieldPath, err)
		}

		jsonPointers := []string{fieldJsonPointer}
		if targetName != "" {
			targetPrefix := "/targets/" + targetName
			jsonPointers = append(jsonPointers, targetPrefix+fieldJsonPointer)
		}

		success := false
		var lastErr error
		var parentNodesToCreate []parentNode

		for _, jsonPointer := range jsonPointers {
			path, err := yamlpatch.ParsePath(jsonPointer)
			if err != nil {
				return "", fmt.Errorf("failed to parse JSON Pointer %s: %w", jsonPointer, err)
			}

			patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))
			var modifiedContent []byte
			var patchErr error

			switch configChange.Operation {
			case OperationRemove:
				modifiedContent, patchErr = patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
					Type: yamlpatch.OperationRemove,
					Path: path,
				}})
			case OperationReplace:
				modifiedContent, patchErr = patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
					Type:  yamlpatch.OperationReplace,
					Path:  path,
					Value: configChange.Value,
				}})
			case OperationAdd:
				modifiedContent, patchErr = patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
					Type:  yamlpatch.OperationAdd,
					Path:  path,
					Value: configChange.Value,
				}})

				// Collect parent path errors for later retry
				if patchErr != nil && isParentPathError(patchErr) {
					if missingPath, extractErr := extractMissingPath(patchErr); extractErr == nil {
						parentNodesToCreate = append(parentNodesToCreate, parentNode{path, missingPath})
					}
				}
			default:
				return "", fmt.Errorf("unknown operation type %q for field %s", configChange.Operation, fieldPath)
			}

			if patchErr == nil {
				content = modifiedContent
				log.Debugf(ctx, "Applied changes to %s", jsonPointer)
				success = true
				lastErr = nil
				break
			}

			log.Debugf(ctx, "Failed to apply change to %s: %v", jsonPointer, patchErr)
			lastErr = patchErr
		}

		// If all attempts failed with parent path errors, try creating nested structures
		if !success && len(parentNodesToCreate) > 0 {
			for _, errInfo := range parentNodesToCreate {
				nestedValue := buildNestedMaps(errInfo.path, errInfo.missingPath, configChange.Value)

				patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))
				modifiedContent, patchErr := patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
					Type:  yamlpatch.OperationAdd,
					Path:  errInfo.missingPath,
					Value: nestedValue,
				}})

				if patchErr == nil {
					content = modifiedContent
					lastErr = nil
					log.Debugf(ctx, "Created nested structure at %s", errInfo.missingPath.String())
					break
				}
				lastErr = patchErr
				log.Debugf(ctx, "Failed to create nested structure at %s: %v", errInfo.missingPath.String(), patchErr)
			}
		}

		if lastErr != nil {
			if (configChange.Operation == OperationRemove || configChange.Operation == OperationReplace) && isPathNotFoundError(lastErr) {
				return "", fmt.Errorf("failed to apply change %s: field not found in YAML configuration: %w", fieldJsonPointer, lastErr)
			}
			return "", fmt.Errorf("failed to apply change %s: %w", fieldJsonPointer, lastErr)
		}

	}

	return string(content), nil
}

// getResolvedFieldChanges builds a map from file paths to lists of field changes
func getResolvedFieldChanges(ctx context.Context, b *bundle.Bundle, configChanges map[string]map[string]*ConfigChangeDesc) (map[string]resolvedChanges, error) {
	resolvedChangesByFile := make(map[string]resolvedChanges)

	for resourceKey, resourceChanges := range configChanges {
		for fieldPath, configChange := range resourceChanges {
			fullPath := resourceKey + "." + fieldPath

			resolvedPath, err := resolveSelectors(fullPath, b)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve selectors in path %s: %w", fullPath, err)
			}

			loc := b.Config.GetLocation(resolvedPath)
			filePath := loc.File

			// If field has no location, find the parent resource's location to then add a new field
			if filePath == "" {
				resourceLocation := b.Config.GetLocation(resourceKey)
				filePath = resourceLocation.File
				if filePath == "" {
					return nil, fmt.Errorf("failed to find location for resource %s for a field %s", resourceKey, fieldPath)
				}

				log.Debugf(ctx, "Field %s has no location, using resource location: %s", fullPath, filePath)
			}

			if _, ok := resolvedChangesByFile[filePath]; !ok {
				resolvedChangesByFile[filePath] = make(resolvedChanges)
			}
			resolvedChangesByFile[filePath][resolvedPath] = configChange
		}
	}

	return resolvedChangesByFile, nil
}

// isParentPathError checks if error indicates missing parent path.
func isParentPathError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "parent path") && strings.Contains(msg, "does not exist")
}

// isPathNotFoundError checks if error indicates the path itself does not exist.
func isPathNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "does not exist")
}

// extractMissingPath extracts the missing path from error message like:
// "op add /a/b/c/d: parent path /a/b does not exist"
// Returns: "/a/b"
func extractMissingPath(err error) (yamlpatch.Path, error) {
	msg := err.Error()
	start := strings.Index(msg, "parent path ")
	if start == -1 {
		return nil, errors.New("could not find 'parent path' in error message")
	}
	start += len("parent path ")

	end := strings.Index(msg[start:], " does not exist")
	if end == -1 {
		return nil, errors.New("could not find 'does not exist' in error message")
	}

	pathStr := msg[start : start+end]
	return yamlpatch.ParsePath(pathStr)
}

// buildNestedMaps creates a nested map structure from targetPath to missingPath.
// Example:
//
//	targetPath: /a/b/c/d/e
//	missingPath: /a/b
//	leafValue: "foo"
//
// Returns: {c: {d: {e: "foo"}}}
func buildNestedMaps(targetPath, missingPath yamlpatch.Path, leafValue any) any {
	missingLen := len(missingPath)
	targetLen := len(targetPath)

	if missingLen >= targetLen {
		// Missing path is not a parent of target path
		return leafValue
	}

	// Build nested structure from leaf to missing parent
	result := leafValue
	for i := targetLen - 1; i >= missingLen; i-- {
		result = map[string]any{
			targetPath[i]: result,
		}
	}

	return result
}

// strPathToJSONPointer converts a structpath string to JSON Pointer format.
// Example: "resources.jobs.test[0].name" -> "/resources/jobs/test/0/name"
func strPathToJSONPointer(pathStr string) (string, error) {
	node, err := structpath.Parse(pathStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse path %q: %w", pathStr, err)
	}

	var parts []string
	for _, n := range node.AsSlice() {
		if key, ok := n.StringKey(); ok {
			parts = append(parts, key)
			continue
		}

		if idx, ok := n.Index(); ok {
			parts = append(parts, strconv.Itoa(idx))
			continue
		}

		return "", fmt.Errorf("unsupported path node type in path %q", pathStr)
	}

	if len(parts) == 0 {
		return "", nil
	}
	return "/" + strings.Join(parts, "/"), nil
}
