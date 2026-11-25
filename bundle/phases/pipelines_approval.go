package phases

import (
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
)

func init() {
	registerDestroyApprovalHook("pipelines", pipelineDestroyApproval)
}

// pipelineDestroyApproval splits the pipeline deletions by cascade_on_destroy, so that the
// message says whether the datasets of a pipeline are deleted as well.
func pipelineDestroyApproval(b *bundle.Bundle, deletes []deployplan.Action) ([]approvalListing, error) {
	var cascading, retaining []deployplan.Action
	for _, a := range deletes {
		cascade, err := pipelineDeletionCascades(b, a)
		if err != nil {
			return nil, err
		}
		if cascade {
			cascading = append(cascading, a)
		} else {
			retaining = append(retaining, a)
		}
	}

	return []approvalListing{
		{resources.PipelineDestroyWithCascadeMessage, cascading},
		{resources.PipelineDestroyNoCascadeMessage, retaining},
	}, nil
}

// pipelineDeletionCascades reports whether deleting the pipeline referenced by a delete action
// also deletes its datasets (MVs, STs, Views). This is the server default (cascade) unless
// cascade_on_destroy is explicitly set to false.
//
// The value is read from the persisted state.
func pipelineDeletionCascades(b *bundle.Bundle, action deployplan.Action) (bool, error) {
	entry, ok := b.DeploymentBundle.StateDB.GetResourceEntry(action.ResourceKey)
	if !ok || len(entry.State) == 0 {
		return true, nil
	}

	var state dresources.PipelineState
	if err := json.Unmarshal(entry.State, &state); err != nil {
		return false, fmt.Errorf("parsing persisted state for %s: %w", action.ResourceKey, err)
	}
	if state.CascadeOnDestroy == nil {
		return true, nil
	}
	return *state.CascadeOnDestroy, nil
}
