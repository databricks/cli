package configsync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/marshal"
	"github.com/palantir/pkg/yamlpatch/gopkgv3yamlpatcher"
	"github.com/palantir/pkg/yamlpatch/yamlpatch"
)

type resolvedChanges map[string]*deployplan.ChangeDesc

// normalizeValue converts values to plain Go types suitable for YAML patching
// by using SDK marshaling which properly handles ForceSendFields and other annotations.
func normalizeValue(v any) (any, error) {
	if v == nil {
		return nil, nil
	}

	switch v.(type) {
	case bool, string, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return v, nil
	}

	rv := reflect.ValueOf(v)
	rt := rv.Type()

	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}

	var data []byte
	var err error

	if rt.Kind() == reflect.Struct {
		data, err = marshal.Marshal(v)
	} else {
		data, err = json.Marshal(v)
	}

	if err != nil {
		return v, fmt.Errorf("failed to marshal value of type %T: %w", v, err)
	}

	var normalized any
	err = json.Unmarshal(data, &normalized)
	if err != nil {
		return v, fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return normalized, nil
}

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
		changeDesc := changes[fieldPath]
		jsonPointer, err := strPathToJSONPointer(fieldPath)
		if err != nil {
			return "", fmt.Errorf("failed to convert field path %q to JSON pointer: %w", fieldPath, err)
		}

		jsonPointers := []string{jsonPointer}
		if targetName != "" {
			targetPrefix := "/targets/" + targetName
			jsonPointers = append(jsonPointers, targetPrefix+jsonPointer)
		}

		hasConfigValue := changeDesc.Old != nil || changeDesc.New != nil
		isRemoval := changeDesc.Remote == nil && hasConfigValue
		isReplacement := changeDesc.Remote != nil && hasConfigValue
		isAddition := changeDesc.Remote != nil && !hasConfigValue

		success := false
		var lastErr error
		var lastPointer string
		var parentNodes []parentNode

		normalizedRemote, err := normalizeValue(changeDesc.Remote)
		if err != nil {
			return "", fmt.Errorf("failed to normalize remote value for %s: %w", jsonPointer, err)
		}

		for _, jsonPointer := range jsonPointers {
			path, err := yamlpatch.ParsePath(jsonPointer)
			if err != nil {
				return "", fmt.Errorf("failed to parse JSON Pointer %s: %w", jsonPointer, err)
			}

			patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))
			var modifiedContent []byte
			var patchErr error

			if isRemoval {
				modifiedContent, patchErr = patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
					Type: yamlpatch.OperationRemove,
					Path: path,
				}})
			} else if isReplacement {
				modifiedContent, patchErr = patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
					Type:  yamlpatch.OperationReplace,
					Path:  path,
					Value: normalizedRemote,
				}})
			} else if isAddition {
				modifiedContent, patchErr = patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
					Type:  yamlpatch.OperationAdd,
					Path:  path,
					Value: normalizedRemote,
				}})

				// Collect parent path errors for later retry
				if patchErr != nil && isParentPathError(patchErr) {
					if missingPath, extractErr := extractMissingPath(patchErr); extractErr == nil {
						parentNodes = append(parentNodes, parentNode{path, missingPath})
					}
				}
			} else {
				return "", fmt.Errorf("unknown operation type for field %s", fieldPath)
			}

			if patchErr == nil {
				content = modifiedContent
				log.Debugf(ctx, "Applied changes to %s", jsonPointer)
				success = true
				break
			}

			log.Debugf(ctx, "Failed to apply change to %s: %v", jsonPointer, patchErr)
			lastErr = patchErr
			lastPointer = jsonPointer
		}

		// If all attempts failed with parent path errors, try creating nested structures
		if !success && len(parentNodes) > 0 {
			for _, errInfo := range parentNodes {
				nestedValue := buildNestedStructure(errInfo.path, errInfo.missingPath, normalizedRemote)

				patcher := gopkgv3yamlpatcher.New(gopkgv3yamlpatcher.IndentSpaces(2))
				modifiedContent, patchErr := patcher.Apply(content, yamlpatch.Patch{yamlpatch.Operation{
					Type:  yamlpatch.OperationAdd,
					Path:  errInfo.missingPath,
					Value: nestedValue,
				}})

				if patchErr == nil {
					content = modifiedContent
					log.Debugf(ctx, "Created nested structure at %s", errInfo.missingPath.String())
					success = true
					break
				}
				log.Debugf(ctx, "Failed to create nested structure at %s: %v", errInfo.missingPath.String(), patchErr)
			}
		}

		if !success {
			if lastErr != nil {
				return "", fmt.Errorf("failed to apply change %s: %w", lastPointer, lastErr)
			}
			return "", fmt.Errorf("failed to apply change for field %s: no valid target found", fieldPath)
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
			resolvedChangesByFile[filePath][resolvedPath] = changeDesc
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

// buildNestedStructure creates a nested map structure from targetPath to missingPath.
// Example:
//
//	targetPath: /a/b/c/d/e
//	missingPath: /a/b
//	leafValue: "foo"
//
// Returns: {c: {d: {e: "foo"}}}
func buildNestedStructure(targetPath, missingPath yamlpatch.Path, leafValue any) any {
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
