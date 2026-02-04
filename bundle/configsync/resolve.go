package configsync

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/structs/structpath"
)

type FieldChange struct {
	FilePath        string
	Change          *ConfigChangeDesc
	FieldCandidates []string
}

// resolveSelectors converts key-value selectors to numeric indices.
// Example: "resources.jobs.foo.tasks[task_key='main'].name" -> "resources.jobs.foo.tasks[1].name"
func resolveSelectors(pathStr string, b *bundle.Bundle, operation OperationType) (*structpath.PathNode, error) {
	node, err := structpath.Parse(pathStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	nodes := node.AsSlice()
	var result *structpath.PathNode
	currentValue := b.Config.Value()

	for _, n := range nodes {
		if key, ok := n.StringKey(); ok {
			result = structpath.NewStringKey(result, key)
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Key(key)})
			}
			continue
		}

		if idx, ok := n.Index(); ok {
			result = structpath.NewIndex(result, idx)
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Index(idx)})
			}
			continue
		}

		// Check for key-value selector: [key='value']
		if key, value, ok := n.KeyValue(); ok {
			if !currentValue.IsValid() || currentValue.Kind() != dyn.KindSequence {
				return nil, fmt.Errorf("cannot apply [%s='%s'] selector to non-array value in path %s", key, value, pathStr)
			}

			seq, _ := currentValue.AsSequence()
			foundIndex := -1

			for i, elem := range seq {
				keyValue, err := dyn.GetByPath(elem, dyn.Path{dyn.Key(key)})
				if err != nil {
					continue
				}

				if keyValue.Kind() == dyn.KindString && keyValue.MustString() == value {
					foundIndex = i
					break
				}
			}

			if foundIndex == -1 {
				if operation == OperationAdd {
					result = structpath.NewBracketStar(result)
					// Can't navigate further into non-existent element
					currentValue = dyn.Value{}
					continue
				}
				return nil, fmt.Errorf("no array element found with %s='%s' in path %s", key, value, pathStr)
			}

			result = structpath.NewIndex(result, foundIndex)
			currentValue = seq[foundIndex]
			continue
		}

		if n.DotStar() || n.BracketStar() {
			return nil, errors.New("wildcard patterns are not supported in field paths")
		}
	}

	return result, nil
}

// pathDepth returns the number of segments in a struct path.
// Used for sorting changes so deeper (leaf) changes are applied before shallower (structural) changes.
func pathDepth(pathStr string) int {
	node, err := structpath.Parse(pathStr)
	if err != nil {
		return 0
	}
	return len(node.AsSlice())
}

// adjustArrayIndex adjusts the index in a PathNode based on previous operations.
// When operations are applied sequentially, removals and additions shift array indices.
// This function adjusts the index to account for those shifts.
func adjustArrayIndex(path *structpath.PathNode, operations map[string][]struct {
	index     int
	operation OperationType
},
) *structpath.PathNode {
	originalIndex, ok := path.Index()
	if !ok {
		return path
	}

	parentPath := path.Parent()
	parentPathStr := parentPath.String()
	ops := operations[parentPathStr]

	// Calculate adjustment based on operations at lower indices
	// Only operations at indices < originalIndex affect this operation
	adjustment := 0
	for _, op := range ops {
		if op.index < originalIndex {
			switch op.operation {
			case OperationRemove:
				adjustment-- // Removals shift indices down
			case OperationAdd:
				adjustment++ // Additions shift indices up
			default:
				// Other operations (Replace, Unknown, Skip) don't affect array structure
			}
		}
	}

	adjustedIndex := originalIndex + adjustment
	if adjustedIndex < 0 {
		adjustedIndex = 0
	}

	return structpath.NewIndex(parentPath, adjustedIndex)
}

// ResolveChanges resolves selectors and computes field path candidates for each change.
func ResolveChanges(ctx context.Context, b *bundle.Bundle, configChanges Changes) ([]FieldChange, error) {
	var result []FieldChange
	targetName := b.Config.Bundle.Target

	resourceKeys := make([]string, 0, len(configChanges))
	for resourceKey := range configChanges {
		resourceKeys = append(resourceKeys, resourceKey)
	}
	sort.Strings(resourceKeys)

	for _, resourceKey := range resourceKeys {
		resourceChanges := configChanges[resourceKey]

		fieldPaths := make([]string, 0, len(resourceChanges))
		for fieldPath := range resourceChanges {
			fieldPaths = append(fieldPaths, fieldPath)
		}

		// Sort field paths by depth (deeper first), then operation type (removals before adds), then alphabetically
		sort.SliceStable(fieldPaths, func(i, j int) bool {
			depthI := pathDepth(fieldPaths[i])
			depthJ := pathDepth(fieldPaths[j])

			// If depths differ, sort by depth descending (deeper paths first)
			if depthI != depthJ {
				return depthI > depthJ
			}

			// Within same depth, sort removals before adding to be able fill removed indices
			// and minimize the diff in cases when the task is recreated because of renaming
			opI := resourceChanges[fieldPaths[i]].Operation
			opJ := resourceChanges[fieldPaths[j]].Operation

			if opI == OperationRemove && opJ != OperationRemove {
				return true
			}
			if opI != OperationRemove && opJ == OperationRemove {
				return false
			}

			// Otherwise maintain alphabetical order for determinism
			return fieldPaths[i] < fieldPaths[j]
		})

		// Create indices map for this resource, path -> indices that we could use to replace with added elements
		indicesToReplaceMap := make(map[string][]int)

		indexOperations := make(map[string][]struct {
			index     int
			operation OperationType
		})

		for _, fieldPath := range fieldPaths {
			configChange := resourceChanges[fieldPath]
			fullPath := resourceKey + "." + fieldPath

			resolvedPath, err := resolveSelectors(fullPath, b, configChange.Operation)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve selectors in path %s: %w", fullPath, err)
			}

			// If the element is removed, we can use the index to replace it with added element
			// That may improve the diff in cases when the task is recreated because of renaming
			if configChange.Operation == OperationRemove {
				freeIndex, ok := resolvedPath.Index()
				if ok {
					parentPath := resolvedPath.Parent().String()
					indicesToReplaceMap[parentPath] = append(indicesToReplaceMap[parentPath], freeIndex)
				}
			}

			if configChange.Operation == OperationAdd && resolvedPath.BracketStar() {
				parentPath := resolvedPath.Parent().String()
				indices, ok := indicesToReplaceMap[parentPath]
				if ok && len(indices) > 0 {
					index := indices[0]
					indicesToReplaceMap[parentPath] = indices[1:]
					resolvedPath = structpath.NewIndex(resolvedPath.Parent(), index)
				}
			}

			// Adjust array indices based on previous operations
			// Operations use indices from the original state, but when applied sequentially,
			// removals and additions shift indices. We adjust here before creating the path string.
			resolvedPath = adjustArrayIndex(resolvedPath, indexOperations)

			// Track this operation for future index adjustments (only for array element operations)
			if originalIndex, ok := resolvedPath.Index(); ok {
				parentPath := resolvedPath.Parent().String()
				indexOperations[parentPath] = append(indexOperations[parentPath], struct {
					index     int
					operation OperationType
				}{originalIndex, configChange.Operation})
			}

			resolvedPathStr := resolvedPath.String()
			candidates := []string{resolvedPathStr}
			if targetName != "" {
				targetPrefixedPath := "targets." + targetName + "." + resolvedPathStr
				candidates = append(candidates, targetPrefixedPath)
			}

			loc := b.Config.GetLocation(resolvedPathStr)
			filePath := loc.File

			isDefinedInConfig := filePath != ""
			if !isDefinedInConfig {
				if configChange.Operation == OperationRemove {
					// If the field is not defined in the config and the operation is remove, it is more likely a CLI default
					// in this case we skip the change
					continue
				}

				if configChange.Operation == OperationReplace {
					// If the field is not defined in the config and the operation is replace, it is more likely a CLI default
					// in this case we add it explicitly to the resource location
					configChange.Operation = OperationAdd
				}

				resourceLocation := b.Config.GetLocation(resourceKey)
				filePath = resourceLocation.File
				if filePath == "" {
					return nil, fmt.Errorf("failed to find location for resource %s for a field %s", resourceKey, fieldPath)
				}

				log.Debugf(ctx, "Field %s has no location, using resource location: %s", fullPath, filePath)
			}

			result = append(result, FieldChange{
				FilePath:        filePath,
				Change:          configChange,
				FieldCandidates: candidates,
			})
		}
	}

	return result, nil
}
