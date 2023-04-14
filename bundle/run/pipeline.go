package run

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/bricks/bundle/run/pipeline"
	"github.com/databricks/bricks/libs/cmdio"
	"github.com/databricks/bricks/libs/flags"
	"github.com/databricks/bricks/libs/log"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	flag "github.com/spf13/pflag"
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
	res, err := w.Pipelines.Impl().ListPipelineEvents(ctx, pipelines.ListPipelineEvents{
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

// PipelineOptions defines options for running a pipeline update.
type PipelineOptions struct {
	// Perform a full graph update.
	RefreshAll bool

	// List of tables to update.
	Refresh []string

	// Perform a full graph reset and recompute.
	FullRefreshAll bool

	// List of tables to reset and recompute.
	FullRefresh []string
}

func (o *PipelineOptions) Define(fs *flag.FlagSet) {
	fs.BoolVar(&o.RefreshAll, "refresh-all", false, "Perform a full graph update.")
	fs.StringSliceVar(&o.Refresh, "refresh", nil, "List of tables to update.")
	fs.BoolVar(&o.FullRefreshAll, "full-refresh-all", false, "Perform a full graph reset and recompute.")
	fs.StringSliceVar(&o.FullRefresh, "full-refresh", nil, "List of tables to reset and recompute.")
}

// Validate returns if the combination of options is valid.
func (o *PipelineOptions) Validate() error {
	set := []string{}
	if o.RefreshAll {
		set = append(set, "--refresh-all")
	}
	if len(o.Refresh) > 0 {
		set = append(set, "--refresh")
	}
	if o.FullRefreshAll {
		set = append(set, "--full-refresh-all")
	}
	if len(o.FullRefresh) > 0 {
		set = append(set, "--full-refresh")
	}
	if len(set) > 1 {
		return fmt.Errorf("pipeline run arguments are mutually exclusive (got %s)", strings.Join(set, ", "))
	}
	return nil
}

func (o *PipelineOptions) toPayload(pipelineID string) (*pipelines.StartUpdate, error) {
	if err := o.Validate(); err != nil {
		return nil, err
	}
	payload := &pipelines.StartUpdate{
		PipelineId: pipelineID,

		// Note: `RefreshAll` is implied if the fields below are not set.
		RefreshSelection:     o.Refresh,
		FullRefresh:          o.FullRefreshAll,
		FullRefreshSelection: o.FullRefresh,
	}
	return payload, nil
}

type pipelineRunner struct {
	key

	bundle   *bundle.Bundle
	pipeline *resources.Pipeline
}

func (r *pipelineRunner) Run(ctx context.Context, opts *Options) (RunOutput, error) {
	var pipelineID = r.pipeline.ID

	// Include resource key in logger.
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("resource", r.Key()))
	w := r.bundle.WorkspaceClient()
	_, err := w.Pipelines.GetByPipelineId(ctx, pipelineID)
	if err != nil {
		log.Warnf(ctx, "Cannot get pipeline: %s", err)
		return nil, err
	}

	req, err := opts.Pipeline.toPayload(pipelineID)
	if err != nil {
		return nil, err
	}

	res, err := w.Pipelines.StartUpdate(ctx, *req)
	if err != nil {
		return nil, err
	}

	updateID := res.UpdateId

	// setup progress logger and tracker to query events
	updateTracker := pipeline.NewUpdateTracker(pipelineID, updateID, w)
	progressLogger, ok := cmdio.FromContext(ctx)
	if !ok {
		return nil, fmt.Errorf("no progress logger found")
	}
	// Inplace logger mode is not supported for pipelines right now
	if progressLogger.Mode == flags.ModeInplace {
		progressLogger.Mode = flags.ModeAppend
	}

	// Log the pipeline update URL as soon as it is available.
	progressLogger.Log(pipeline.NewUpdateUrlEvent(w.Config.Host, updateID, pipelineID))

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
