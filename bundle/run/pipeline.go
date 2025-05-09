package run

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/bundle/run/progress"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

func filterEventsByUpdateId(events []pipelines.PipelineEvent, updateId string) []pipelines.PipelineEvent {
	var result []pipelines.PipelineEvent
	for i := range events {
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
		for i := range len(event.Error.Exceptions) {
			logString += event.Error.Exceptions[i].Message + "\n"
		}
	}
	if logString != "" {
		log.Errorf(ctx, "[%s] %s", event.EventType, logString)
	}
}

func (r *pipelineRunner) logErrorEvent(ctx context.Context, pipelineId, updateId string) error {
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
	events, err := w.Pipelines.ListPipelineEventsAll(ctx, pipelines.ListPipelineEventsRequest{
		Filter:     `level='ERROR'`,
		MaxResults: 100,
		PipelineId: pipelineId,
	})
	if err != nil {
		return err
	}
	updateEvents := filterEventsByUpdateId(events, updateId)
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
	if r.pipeline == nil {
		return ""
	}
	return r.pipeline.CreatePipeline.Name
}

func (r *pipelineRunner) Run(ctx context.Context, opts *Options) (output.RunOutput, error) {
	pipelineID := r.pipeline.ID

	// Include resource key in logger.
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("resource", r.Key()))
	w := r.bundle.WorkspaceClient()

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
		return nil, errors.New("no progress logger found")
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
			log.Info(ctx, event.String())
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
			return nil, errors.New("update cancelled")
		}
		if state == pipelines.UpdateInfoStateFailed {
			log.Infof(ctx, "Update has failed!")
			err := r.logErrorEvent(ctx, pipelineID, updateID)
			if err != nil {
				return nil, err
			}
			return nil, errors.New("update failed")
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

	// Waits for the Idle state of the pipeline
	_, err = wait.GetWithTimeout(jobRunTimeout)
	return err
}

func (r *pipelineRunner) Restart(ctx context.Context, opts *Options) (output.RunOutput, error) {
	s := cmdio.Spinner(ctx)
	s <- "Cancelling the active pipeline update"
	err := r.Cancel(ctx)
	close(s)
	if err != nil {
		return nil, err
	}

	return r.Run(ctx, opts)
}

func (r *pipelineRunner) ParseArgs(args []string, opts *Options) error {
	if len(args) == 0 {
		return nil
	}

	return fmt.Errorf("received %d unexpected positional arguments", len(args))
}

func (r *pipelineRunner) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}
