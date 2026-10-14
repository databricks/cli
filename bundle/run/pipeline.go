package run

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/bundle/run/progress"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/sdk-go/options/lro"
	sdkpipelines "github.com/databricks/sdk-go/pipelines/v2"
	"github.com/spf13/cobra"
)

func filterEventsByUpdateId(events []pipelines.PipelineEvent, updateId string) []pipelines.PipelineEvent {
	var result []pipelines.PipelineEvent
	for i := range events {
		if events[i].Origin != nil && utils.Deref(events[i].Origin.UpdateId) == updateId {
			result = append(result, events[i])
		}
	}
	return result
}

func (r *pipelineRunner) logEvent(ctx context.Context, event pipelines.PipelineEvent) {
	var sb strings.Builder
	if message := utils.Deref(event.Message); message != "" {
		fmt.Fprintf(&sb, " %s\n", message)
	}
	if event.Error != nil && len(event.Error.Exceptions) > 0 {
		sb.WriteString("trace for most recent exception: \n")
		for _, exc := range event.Error.Exceptions {
			sb.WriteString(utils.Deref(exc.Message))
			sb.WriteByte('\n')
		}
	}
	if sb.Len() > 0 {
		log.Errorf(ctx, "[%s] %s", utils.Deref(event.EventType), sb.String())
	}
}

func (r *pipelineRunner) logErrorEvent(ctx context.Context, pipelineId, updateId string) error {
	w := r.bundle.WorkspaceClient(ctx)

	// Note: For a 100 percent correct and complete solution we should use the
	// w.Pipelines.ListPipelineEventsAll method to find all relevant events. However the
	// probablity of the relevant last error event not being present in the most
	// recent 100 error events is very close to 0 and the first 100 error events
	// should give us a good picture of the error.
	//
	// Otherwise for long lived pipelines, there can be a lot of unnecessary
	// latency due to multiple pagination API calls needed underneath the hood for
	// ListPipelineEventsAll
	events, err := progress.ListEvents(ctx, w, sdkpipelines.ListPipelineEventsRequest{
		Filter:     new(`level='ERROR'`),
		MaxResults: new(100),
		PipelineId: new(pipelineId),
	})
	if err != nil {
		return err
	}
	updateEvents := filterEventsByUpdateId(events, updateId)
	// The events API returns most recent events first.
	for _, event := range slices.Backward(updateEvents) {
		r.logEvent(ctx, event)
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
	return utils.Deref(r.pipeline.Name)
}

func (r *pipelineRunner) Run(ctx context.Context, opts *Options) (output.RunOutput, error) {
	pipelineID := r.pipeline.ID

	// Include resource key in logger.
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("resource", r.Key()))
	w := r.bundle.WorkspaceClient(ctx)

	// The preset is defaulted to true by "mode: development" and honors an explicit opt-out.
	development := config.IsExplicitlyEnabled(r.bundle.Config.Presets.PipelinesDevelopment)

	req, err := opts.Pipeline.toPayload(r.pipeline, pipelineID, development)
	if err != nil {
		return nil, err
	}

	res, err := w.Pipelines.Start(ctx, *req)
	if err != nil {
		return nil, err
	}

	updateID := utils.Deref(res.UpdateId)

	// setup progress logger and tracker to query events
	updateTracker := progress.NewUpdateTracker(pipelineID, updateID, w)

	// Log the pipeline update URL as soon as it is available.
	cmdio.Log(ctx, progress.NewPipelineUpdateUrlEvent(w.Config.Host, updateID, pipelineID))

	if opts.NoWait {
		return &output.PipelineOutput{
			UpdateId: updateID,
		}, nil
	}

	// Poll update for completion and post status.
	// Note: there is no "StartUpdateAndWait" wrapper for this API.
	var prevState *pipelines.UpdateState
	for {
		events, err := updateTracker.Events(ctx)
		if err != nil {
			return nil, err
		}
		for _, event := range events {
			cmdio.Log(ctx, &event)
			log.Info(ctx, event.String())
		}

		update, err := w.Pipelines.GetUpdate(ctx, sdkpipelines.GetUpdateRequest{
			PipelineId: new(pipelineID),
			UpdateId:   new(updateID),
		})
		if err != nil {
			return nil, err
		}

		// Log only if the current state is different from the previous state.
		var state pipelines.UpdateState
		if update.Update != nil {
			state = update.Update.State
		}
		if prevState == nil || *prevState != state {
			log.Infof(ctx, "Update status: %s", state)
			prevState = &state
		}

		if state == pipelines.UpdateState_Canceled {
			log.Infof(ctx, "Update was cancelled!")
			return nil, errors.New("update cancelled")
		}
		if state == pipelines.UpdateState_Failed {
			log.Infof(ctx, "Update has failed!")
			err := r.logErrorEvent(ctx, pipelineID, updateID)
			if err != nil {
				return nil, err
			}
			return nil, errors.New("update failed")
		}
		if state == pipelines.UpdateState_Completed {
			log.Infof(ctx, "Update has completed successfully!")
			return &output.PipelineOutput{
				UpdateId: updateID,
			}, nil
		}

		time.Sleep(time.Second)
	}
}

func (r *pipelineRunner) Cancel(ctx context.Context) error {
	w := r.bundle.WorkspaceClient(ctx)
	wait, err := w.Pipelines.Stop(ctx, sdkpipelines.StopPipelineRequest{
		PipelineId: new(r.pipeline.ID),
	})
	if err != nil {
		return err
	}

	// Waits for the Idle state of the pipeline
	_, err = wait.Wait(ctx, lro.WithTimeout(jobRunTimeout))
	return err
}

func (r *pipelineRunner) Restart(ctx context.Context, opts *Options) (output.RunOutput, error) {
	sp := cmdio.NewSpinner(ctx)
	sp.Update("Cancelling the active pipeline update")
	err := r.Cancel(ctx)
	sp.Close()
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
