package run

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/bundle/run/progress"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

func filterEventsByUpdateId(events []pipelines.PipelineEvent, updateId string) []pipelines.PipelineEvent {
	result := []pipelines.PipelineEvent{}
	for i := 0; i < len(events); i++ {
		if events[i].Origin.UpdateId == updateId {
			result = append(result, events[i])
		}
	}
	return result
}

func (r *pipelineRunner) logEvent(ctx context.Context, event pipelines.PipelineEvent) {
	logString := ""
	if event.Message != "" {
		logString += fmt.Sprintf(" %s\n", event.Message)
	}
	if event.Error != nil && len(event.Error.Exceptions) > 0 {
		logString += "trace for most recent exception: \n"
		for i := 0; i < len(event.Error.Exceptions); i++ {
			logString += fmt.Sprintf("%s\n", event.Error.Exceptions[i].Message)
		}
	}
	if logString != "" {
		log.Errorf(ctx, fmt.Sprintf("[%s] %s", event.EventType, logString))
	}
}

func (r *pipelineRunner) logErrorEvent(ctx context.Context, pipelineId string, updateId string) error {
	w := r.bundle.WorkspaceClient()

	// Note: For a 100 percent correct and complete solution we should use the
	// w.Pipelines.ListPipelineEventsAll method to find all relevant events. However the
	// probablity of the relevant last error event not being present in the most
	// recent 100 error events is very close to 0 and the first 100 error events
	// should give us a good picture of the error.
	//
	// Otherwise for long lived pipelines, there can be a lot of unnecessary
	// latency due to multiple pagination API calls needed underneath the hood for
	// ListPipelineEventsAll
	res, err := w.Pipelines.Impl().ListPipelineEvents(ctx, pipelines.ListPipelineEventsRequest{
		Filter:     `level='ERROR'`,
		MaxResults: 100,
		PipelineId: pipelineId,
	})
	if err != nil {
		return err
	}
	updateEvents := filterEventsByUpdateId(res.Events, updateId)
	// The events API returns most recent events first. We iterate in a reverse order
	// to print the events chronologically
	for i := len(updateEvents) - 1; i >= 0; i-- {
		r.logEvent(ctx, updateEvents[i])
	}
	return nil
}

type pipelineRunner struct {
	key

	bundle   *bundle.Bundle
	pipeline *resources.Pipeline
}

func (r *pipelineRunner) Name() string {
	if r.pipeline == nil || r.pipeline.PipelineSpec == nil {
		return ""
	}
	return r.pipeline.PipelineSpec.Name
}

func (r *pipelineRunner) Run(ctx context.Context, opts *Options) (output.RunOutput, error) {
	var pipelineID = r.pipeline.ID

	// Include resource key in logger.
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("resource", r.Key()))
	w := r.bundle.WorkspaceClient()
	_, err := w.Pipelines.GetByPipelineId(ctx, pipelineID)
	if err != nil {
		log.Warnf(ctx, "Cannot get pipeline: %s", err)
		return nil, err
	}

	req, err := opts.Pipeline.toPayload(r.pipeline, pipelineID)
	if err != nil {
		return nil, err
	}

	res, err := w.Pipelines.StartUpdate(ctx, *req)
	if err != nil {
		return nil, err
	}

	updateID := res.UpdateId

	// setup progress logger and tracker to query events
	updateTracker := progress.NewUpdateTracker(pipelineID, updateID, w)
	progressLogger, ok := cmdio.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no progress logger found")
	}

	// Log the pipeline update URL as soon as it is available.
	progressLogger.Log(progress.NewPipelineUpdateUrlEvent(w.Config.Host, updateID, pipelineID))

	if opts.NoWait {
		return nil, nil
	}

	// Poll update for completion and post status.
	// Note: there is no "StartUpdateAndWait" wrapper for this API.
	var prevState *pipelines.UpdateInfoState
	for {
		events, err := updateTracker.Events(ctx)
		if err != nil {
			return nil, err
		}
		for _, event := range events {
			progressLogger.Log(&event)
			log.Infof(ctx, event.String())
		}

		update, err := w.Pipelines.GetUpdateByPipelineIdAndUpdateId(ctx, pipelineID, updateID)
		if err != nil {
			return nil, err
		}

		// Log only if the current state is different from the previous state.
		state := update.Update.State
		if prevState == nil || *prevState != state {
			log.Infof(ctx, "Update status: %s", state)
			prevState = &state
		}

		if state == pipelines.UpdateInfoStateCanceled {
			log.Infof(ctx, "Update was cancelled!")
			return nil, fmt.Errorf("update cancelled")
		}
		if state == pipelines.UpdateInfoStateFailed {
			log.Infof(ctx, "Update has failed!")
			err := r.logErrorEvent(ctx, pipelineID, updateID)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("update failed")
		}
		if state == pipelines.UpdateInfoStateCompleted {
			log.Infof(ctx, "Update has completed successfully!")
			return nil, nil
		}

		time.Sleep(time.Second)
	}
}

func (r *pipelineRunner) Cancel(ctx context.Context) error {
	w := r.bundle.WorkspaceClient()
	wait, err := w.Pipelines.Stop(ctx, pipelines.StopRequest{
		PipelineId: r.pipeline.ID,
	})

	if err != nil {
		return err
	}

	_, err = wait.GetWithTimeout(jobRunTimeout)
	return err
}
