package configsync

import (
	"context"
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

// resolveSelectors converts key-value selectors to numeric indices that match
// the YAML file positions. It also returns the location of the resolved leaf value.
// Example: "resources.jobs.foo.tasks[task_key='main'].name" -> "resources.jobs.foo.tasks[1].name"
// Returns a PatternNode because for Add operations, [*] may be used as a placeholder for new elements.
func resolveSelectors(pathStr string, b *bundle.Bundle, operation OperationType) (*structpath.PatternNode, dyn.Location, error) {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return nil, dyn.Location{}, fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	nodes := node.AsSlice()
	var result *structpath.PatternNode
	currentValue := b.Config.Value()

	for _, n := range nodes {
		if key, ok := n.StringKey(); ok {
			result = structpath.NewPatternStringKey(result, key)
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Key(key)})
			}
			continue
		}

		if idx, ok := n.Index(); ok {
			result = structpath.NewPatternIndex(result, idx)
			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Index(idx)})
			}
			continue
		}

		// Check for key-value selector: [key='value']
		if key, value, ok := n.KeyValue(); ok {
			if !currentValue.IsValid() || currentValue.Kind() != dyn.KindSequence {
				return nil, dyn.Location{}, fmt.Errorf("cannot apply [%s='%s'] selector to non-array value in path %s", key, value, pathStr)
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
					result = structpath.NewPatternBracketStar(result)
					// Can't navigate further into non-existent element
					currentValue = dyn.Value{}
					continue
				}
				return nil, dyn.Location{}, fmt.Errorf("no array element found with %s='%s' in path %s", key, value, pathStr)
			}

			// Mutators may reorder sequence elements (e.g., tasks sorted by task_key).
			// Use location information to determine the original YAML file position.
			yamlIndex := yamlFileIndex(seq, foundIndex)
			result = structpath.NewPatternIndex(result, yamlIndex)
			currentValue = seq[foundIndex]
			continue
		}
	}

	return result, currentValue.Location(), nil
}

// yamlFileIndex determines the original YAML file position of a sequence element.
// Mutators may reorder sequence elements (e.g., tasks sorted by task_key), so the
// in-memory index may not match the position in the YAML file. This function uses
// location information to count how many elements from the same file appear before
// the target element, giving the correct index for YAML patching.
func yamlFileIndex(seq []dyn.Value, sortedIndex int) int {
	matchLocation := seq[sortedIndex].Location()
	if matchLocation.File == "" {
		return sortedIndex
	}

	yamlIndex := 0
	for i, elem := range seq {
		if i == sortedIndex {
			continue
		}
		loc := elem.Location()
		if loc.File == matchLocation.File && loc.Line < matchLocation.Line {
			yamlIndex++
		}
	}
	return yamlIndex
}

func pathDepth(pathStr string) int {
	node, err := structpath.ParsePath(pathStr)
	if err != nil {
		return 0
	}
	return len(node.AsSlice())
}

// adjustArrayIndex adjusts the index in a PatternNode based on previous operations.
// When operations are applied sequentially, removals and additions shift array indices.
// This function adjusts the index to account for those shifts.
func adjustArrayIndex(path *structpath.PatternNode, operations map[string][]struct {
	index     int
	operation OperationType
},
) *structpath.PatternNode {
	originalIndex, ok := path.Index()
	if !ok {
		return path
	}

	parentPath := path.Parent()
	parentPathStr := parentPath.String()
	ops := operations[parentPathStr]

	adjustment := 0
	for _, op := range ops {
		if op.index < originalIndex {
			switch op.operation {
			case OperationRemove:
				adjustment--
			case OperationAdd:
				adjustment++
			default:
			}
		}
	}

	adjustedIndex := originalIndex + adjustment
	if adjustedIndex < 0 {
		adjustedIndex = 0
	}

	return structpath.NewPatternIndex(parentPath, adjustedIndex)
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
		fieldPathsDepths := map[string]int{}
		for fieldPath := range resourceChanges {
			fieldPaths = append(fieldPaths, fieldPath)
			fieldPathsDepths[fieldPath] = pathDepth(fieldPath)
		}

		// Sort field paths by depth (deeper first), then operation type (removals before adds), then alphabetically
		sort.SliceStable(fieldPaths, func(i, j int) bool {
			depthI := fieldPathsDepths[fieldPaths[i]]
			depthJ := fieldPathsDepths[fieldPaths[j]]

			if depthI != depthJ {
				return depthI > depthJ
			}

			opI := resourceChanges[fieldPaths[i]].Operation
			opJ := resourceChanges[fieldPaths[j]].Operation

			if opI == OperationRemove && opJ != OperationRemove {
				return true
			}
			if opI != OperationRemove && opJ == OperationRemove {
				return false
			}

			return fieldPaths[i] < fieldPaths[j]
		})

		// Create indices map for this resource, path -> indices, that we could use to replace with added elements
		indicesToReplaceMap := make(map[string][]int)

		indexOperations := make(map[string][]struct {
			index     int
			operation OperationType
		})

		for _, fieldPath := range fieldPaths {
			configChange := resourceChanges[fieldPath]
			fullPath := resourceKey + "." + fieldPath

			resolvedPath, resolvedLocation, err := resolveSelectors(fullPath, b, configChange.Operation)
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
					resolvedPath = structpath.NewPatternIndex(resolvedPath.Parent(), index)
				}
			}

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

			filePath := resolvedLocation.File

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
