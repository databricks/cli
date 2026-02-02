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
func resolveSelectors(pathStr string, b *bundle.Bundle, operation OperationType) (string, error) {
	node, err := structpath.Parse(pathStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse path %s: %w", pathStr, err)
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
				return "", fmt.Errorf("cannot apply [%s='%s'] selector to non-array value in path %s", key, value, pathStr)
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
					result = structpath.NewDotStar(result)
					// Can't navigate further into non-existent element
					currentValue = dyn.Value{}
					continue
				}
				return "", fmt.Errorf("no array element found with %s='%s' in path %s", key, value, pathStr)
			}

			result = structpath.NewIndex(result, foundIndex)
			currentValue = seq[foundIndex]
			continue
		}

		if n.DotStar() || n.BracketStar() {
			return "", errors.New("wildcard patterns are not supported in field paths")
		}
	}

	return result.String(), nil
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
		sort.Strings(fieldPaths)

		for _, fieldPath := range fieldPaths {
			configChange := resourceChanges[fieldPath]
			fullPath := resourceKey + "." + fieldPath

			resolvedPath, err := resolveSelectors(fullPath, b, configChange.Operation)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve selectors in path %s: %w", fullPath, err)
			}

			candidates := []string{resolvedPath}
			if targetName != "" {
				targetPrefixedPath := "targets." + targetName + "." + resolvedPath
				candidates = append(candidates, targetPrefixedPath)
			}

			loc := b.Config.GetLocation(resolvedPath)
			filePath := loc.File

			if filePath == "" {
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
