package configsync

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// FormatTextOutput formats the config changes as human-readable text. Useful for debugging
func FormatTextOutput(changes Changes) string {
	var output strings.Builder

	if len(changes) == 0 {
		output.WriteString("No changes detected.\n")
		return output.String()
	}

	fmt.Fprintf(&output, "Detected changes in %d resource(s):\n\n", len(changes))

	resourceKeys := slices.Sorted(maps.Keys(changes))

	for _, resourceKey := range resourceKeys {
		resourceChanges := changes[resourceKey]
		fmt.Fprintf(&output, "Resource: %s\n", resourceKey)

		paths := slices.Sorted(maps.Keys(resourceChanges))

		for _, path := range paths {
			configChange := resourceChanges[path]
			fmt.Fprintf(&output, "  %s: %s\n", path, configChange.Operation)
		}

		output.WriteString("\n")
	}

	return output.String()
}
