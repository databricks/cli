package configsync

import (
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
)

// resolveSelectors converts key-value selectors to numeric indices.
// Example: "tasks[task_key='main'].name" -> "tasks[1].name"
func resolveSelectors(pathStr string, b *bundle.Bundle) (string, error) {
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
