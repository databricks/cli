package phases

import (
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
)

// filterGroup returns actions that match the specified group and any of the specified action types
func filterGroup(changes []deployplan.Action, group string, actionTypes ...deployplan.ActionType) []deployplan.Action {
	var result []deployplan.Action

	// Create a set of action types for efficient lookup
	actionTypeSet := make(map[deployplan.ActionType]bool)
	for _, actionType := range actionTypes {
		actionTypeSet[actionType] = true
	}

	for _, action := range changes {
		actionGroup := config.GetResourceTypeFromKey(action.ResourceKey)
		if actionGroup == group && actionTypeSet[action.ActionType] {
			result = append(result, action)
		}
	}
	return result
}
