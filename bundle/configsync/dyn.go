package configsync

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
)

// resolveSelectors converts key-value selectors to numeric indices.
// Example: "tasks[task_key='main'].name" -> "tasks[1].name"
// Key-value selectors like [key='value'] are resolved by looking up the matching array element.
// Returns error if selector doesn't match any element or is applied to non-array value.
func resolveSelectors(pathStr string, b *bundle.Bundle) (string, error) {
	node, err := structpath.Parse(pathStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	nodes := node.AsSlice()
	var builder strings.Builder
	currentValue := b.Config.Value()

	for i, n := range nodes {
		if key, ok := n.StringKey(); ok {
			if i > 0 {
				builder.WriteString(".")
			}
			builder.WriteString(key)

			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Key(key)})
			}
			continue
		}

		if idx, ok := n.Index(); ok {
			builder.WriteString("[")
			builder.WriteString(strconv.Itoa(idx))
			builder.WriteString("]")

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

			builder.WriteString("[")
			builder.WriteString(strconv.Itoa(foundIndex))
			builder.WriteString("]")
			currentValue = seq[foundIndex]
			continue
		}

		if n.DotStar() || n.BracketStar() {
			return "", errors.New("wildcard patterns are not supported in field paths")
		}
	}

	return builder.String(), nil
}

// structpathToDynPath converts a structpath string to a dyn.Path.
// Expects selectors to be already resolved to numeric indices.
// Example: "tasks[0].timeout_seconds" -> Path{Key("tasks"), Index(0), Key("timeout_seconds")}
func structpathToDynPath(pathStr string) (dyn.Path, error) {
	node, err := structpath.Parse(pathStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	nodes := node.AsSlice()
	var dynPath dyn.Path

	for _, n := range nodes {
		if key, ok := n.StringKey(); ok {
			dynPath = append(dynPath, dyn.Key(key))
			continue
		}

		if idx, ok := n.Index(); ok {
			dynPath = append(dynPath, dyn.Index(idx))
			continue
		}

		// Key-value selectors should be resolved before calling this function
		if key, value, ok := n.KeyValue(); ok {
			return nil, fmt.Errorf("unresolved selector [%s='%s'] in path %s - call resolveSelectors first", key, value, pathStr)
		}

		if n.DotStar() || n.BracketStar() {
			return nil, errors.New("wildcard patterns are not supported in field paths")
		}
	}

	return dynPath, nil
}

// strPathToJSONPointer converts a structpath string to RFC 6902 JSON Pointer format.
// Expects selectors to be already resolved to numeric indices.
// Example: "resources.jobs.test[0].name" -> "/resources/jobs/test/0/name"
func strPathToJSONPointer(pathStr string) (string, error) {
	if pathStr == "" {
		return "", nil
	}

	node, err := structpath.Parse(pathStr)
	if err != nil {
		return "", fmt.Errorf("failed to parse path %s: %w", pathStr, err)
	}

	nodes := node.AsSlice()
	var builder strings.Builder

	for _, n := range nodes {
		if key, ok := n.StringKey(); ok {
			builder.WriteString("/")
			// Escape special characters per RFC 6902
			// ~ must be escaped as ~0
			// / must be escaped as ~1
			escaped := strings.ReplaceAll(key, "~", "~0")
			escaped = strings.ReplaceAll(escaped, "/", "~1")
			builder.WriteString(escaped)
			continue
		}

		if idx, ok := n.Index(); ok {
			builder.WriteString("/")
			builder.WriteString(strconv.Itoa(idx))
			continue
		}

		// Key-value selectors should be resolved before calling this function
		if key, value, ok := n.KeyValue(); ok {
			return "", fmt.Errorf("unresolved selector [%s='%s'] in path %s - call resolveSelectors first", key, value, pathStr)
		}

		if n.DotStar() || n.BracketStar() {
			return "", errors.New("wildcard patterns are not supported in field paths")
		}
	}

	return builder.String(), nil
}

// dynPathToJSONPointer converts a dyn.Path to RFC 6902 JSON Pointer format
// Example: [Key("resources"), Key("jobs"), Key("my_job")] -> "/resources/jobs/my_job"
// Example: [Key("tasks"), Index(1), Key("timeout")] -> "/tasks/1/timeout"
func dynPathToJSONPointer(path dyn.Path) string {
	if len(path) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, component := range path {
		builder.WriteString("/")

		// Handle Key components
		if key := component.Key(); key != "" {
			// Escape special characters per RFC 6902
			// ~ must be escaped as ~0
			// / must be escaped as ~1
			escaped := strings.ReplaceAll(key, "~", "~0")
			escaped = strings.ReplaceAll(escaped, "/", "~1")
			builder.WriteString(escaped)
			continue
		}

		// Handle Index components
		if idx := component.Index(); idx >= 0 {
			builder.WriteString(strconv.Itoa(idx))
			continue
		}
	}

	return builder.String()
}
