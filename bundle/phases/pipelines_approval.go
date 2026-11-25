package phases

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/bundle/direct/dresources"
	"github.com/databricks/cli/libs/cmdio"
)

// logPipelineDeleteApproval prints the pipeline deletions. If cascade_on_destroy is true, we will include
// a note that datasets will be deleted as well.
func logPipelineDeleteApproval(ctx context.Context, b *bundle.Bundle, actions []deployplan.Action, quiet bool) error {
	pipelineDeletes := filterGroup(actions, "pipelines", deployplan.Delete)

	var cascading, retaining []deployplan.Action
	for _, a := range pipelineDeletes {
		cascade, err := pipelineDeletionCascades(b, a)
		if err != nil {
			return err
		}
		if cascade {
			cascading = append(cascading, a)
		} else {
			retaining = append(retaining, a)
		}
	}

	for _, grp := range []struct {
		message string
		actions []deployplan.Action
	}{
		{deletePipelineWithCascadeMessage, cascading},
		{deletePipelineNoCascadeMessage, retaining},
	} {
		if len(grp.actions) == 0 || quiet {
			continue
		}
		cmdio.LogString(ctx, grp.message)
		for _, a := range grp.actions {
			cmdio.Log(ctx, a)
		}
		cmdio.LogString(ctx, "")
	}
	return nil
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
