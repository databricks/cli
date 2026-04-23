package phases

import (
	"strings"

	"github.com/databricks/cli/ucm/deployplan"
)

// filterGroup returns actions whose resource key belongs to the given group
// ("catalogs", "schemas", "volumes", etc.) and whose ActionType is in the
// supplied set. Mirrors bundle/phases.filterGroup but works off the flat
// "resources.<group>.<name>" key convention used by ucm plans.
func filterGroup(actions []deployplan.Action, group string, actionTypes ...deployplan.ActionType) []deployplan.Action {
	actionTypeSet := make(map[deployplan.ActionType]bool, len(actionTypes))
	for _, at := range actionTypes {
		actionTypeSet[at] = true
	}

	var result []deployplan.Action
	for _, a := range actions {
		if !actionTypeSet[a.ActionType] {
			continue
		}
		if resourceGroup(a.ResourceKey) == group {
			result = append(result, a)
		}
	}
	return result
}

// resourceGroup returns the group segment of a resource key
// ("resources.<group>.<name>"), or "" when the key doesn't match that shape.
func resourceGroup(key string) string {
	parts := strings.SplitN(key, ".", 3)
	if len(parts) < 3 || parts[0] != "resources" {
		return ""
	}
	return parts[1]
}
