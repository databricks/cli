package phases

import (
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/service/vectorsearch"
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

// splitVectorSearchIndexActions partitions the given vector_search_indexes
// actions by index_type. Actions for resources not in the bundle config (e.g.
// removed-from-bundle deletes) fall into the "other" bucket where they get a
// generic message instead of a type-specific one.
func splitVectorSearchIndexActions(b *bundle.Bundle, actions []deployplan.Action) (deltaSync, directAccess, other []deployplan.Action) {
	for _, action := range actions {
		switch indexTypeForAction(b, action) {
		case vectorsearch.VectorIndexTypeDeltaSync:
			deltaSync = append(deltaSync, action)
		case vectorsearch.VectorIndexTypeDirectAccess:
			directAccess = append(directAccess, action)
		default:
			other = append(other, action)
		}
	}
	return deltaSync, directAccess, other
}

// indexTypeForAction returns the index_type from the bundle config; an empty
// string means we couldn't determine it (likely the resource was removed from
// the bundle so this is a delete action).
func indexTypeForAction(b *bundle.Bundle, action deployplan.Action) vectorsearch.VectorIndexType {
	const prefix = "resources.vector_search_indexes."
	rest, ok := strings.CutPrefix(action.ResourceKey, prefix)
	if !ok {
		return ""
	}
	idx, ok := b.Config.Resources.VectorSearchIndexes[rest]
	if !ok || idx == nil {
		return ""
	}
	return idx.IndexType
}
