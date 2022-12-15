package run

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type pipelineRunner struct {
	key

	bundle   *bundle.Bundle
	pipeline *resources.Pipeline
}

func (r *pipelineRunner) Run(ctx context.Context) error {
	pipelineID := r.pipeline.ID

	w := r.bundle.WorkspaceClient()
	_, err := w.Pipelines.GetPipelineByPipelineId(ctx, pipelineID)
	if err != nil {
		log.Printf("[WARN] Cannot get pipeline: %s", err)
		return err
	}

	res, err := w.Pipelines.StartUpdate(ctx, pipelines.StartUpdate{
		PipelineId: pipelineID,
	})
	if err != nil {
		return err
	}

	updateID := res.UpdateId

	// Log the pipeline update URL as soon as it is available.
	url := fmt.Sprintf("%s/#joblist/pipelines/%s/updates/%s", w.Config.Host, pipelineID, updateID)
	log.Printf("[INFO] Pipeline update available at %s", url)

	// Poll update for completion and post status.
	// Note: there is no "StartUpdateAndWait" wrapper for this API.
	var prevState *pipelines.UpdateInfoState
	for {
		update, err := w.Pipelines.GetUpdateByPipelineIdAndUpdateId(ctx, pipelineID, updateID)
		if err != nil {
			return err
		}

		// Log only if the current state is different from the previous state.
		state := update.Update.State
		if prevState == nil || *prevState != state {
			log.Printf("[INFO] Pipeline update status: %s", state)
			prevState = &state
		}

		if state == pipelines.UpdateInfoStateCanceled {
			return fmt.Errorf("pipeline state: %s", state)
		}
		if state == pipelines.UpdateInfoStateFailed {
			return fmt.Errorf("pipeline state: %s", state)
		}
		if state == pipelines.UpdateInfoStateCompleted {
			return nil
		}

		time.Sleep(time.Second)
	}
}
