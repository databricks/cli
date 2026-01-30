package configsync

import (
	"fmt"
	"sort"
	"strings"
)

// FormatTextOutput formats the config changes as human-readable text. Useful for debugging
func FormatTextOutput(changes Changes) string {
	var output strings.Builder

	if len(changes) == 0 {
		output.WriteString("No changes detected.\n")
		return output.String()
	}

	output.WriteString(fmt.Sprintf("Detected changes in %d resource(s):\n\n", len(changes)))

	resourceKeys := make([]string, 0, len(changes))
	for key := range changes {
		resourceKeys = append(resourceKeys, key)
	}
	sort.Strings(resourceKeys)

	for _, resourceKey := range resourceKeys {
		resourceChanges := changes[resourceKey]
		output.WriteString(fmt.Sprintf("Resource: %s\n", resourceKey))

		paths := make([]string, 0, len(resourceChanges))
		for path := range resourceChanges {
			paths = append(paths, path)
		}
		sort.Strings(paths)

		for _, path := range paths {
			if resourceChanges[path].Operation == OperationSkip {
				continue
			}

			configChange := resourceChanges[path]
			output.WriteString(fmt.Sprintf("  %s: %s\n", path, configChange.Operation))
		}

		output.WriteString("\n")
	}

	return output.String()
}
