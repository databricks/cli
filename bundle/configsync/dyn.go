package configsync

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/structs/structpath"
)

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
		if key, ok := n.StringKey(); ok {
			dynPath = append(dynPath, dyn.Key(key))

			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Key(key)})
			}
			continue
		}

		if idx, ok := n.Index(); ok {
			dynPath = append(dynPath, dyn.Index(idx))

			if currentValue.IsValid() {
				currentValue, _ = dyn.GetByPath(currentValue, dyn.Path{dyn.Index(idx)})
			}
			continue
		}

		// Check for key-value selector: [key='value']
		if key, value, ok := n.KeyValue(); ok {
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
