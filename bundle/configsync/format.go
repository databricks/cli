package configsync

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle/deployplan"
)

// FormatTextOutput formats the config changes as human-readable text. Useful for debugging
func FormatTextOutput(changes map[string]deployplan.Changes) string {
	var output strings.Builder

	if len(changes) == 0 {
		output.WriteString("No changes detected.\n")
		return output.String()
	}

	output.WriteString(fmt.Sprintf("Detected changes in %d resource(s):\n\n", len(changes)))

	for resourceKey, resourceChanges := range changes {
		output.WriteString(fmt.Sprintf("Resource: %s\n", resourceKey))

		for path, changeDesc := range resourceChanges {
			output.WriteString(fmt.Sprintf("  %s: %s\n", path, changeDesc.Action))
		}
	}

	return output.String()
}
