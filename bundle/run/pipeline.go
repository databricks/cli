package run

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/fatih/color"
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

func (r *pipelineRunner) logEvent(event pipelines.PipelineEvent) {
	red := color.New(color.FgRed).SprintFunc()
	errorPrefix := red("[ERROR]")
	pipelineKeyPrefix := red(fmt.Sprintf("[%s]", r.Key()))
	eventTypePrefix := red(fmt.Sprintf("[%s]", event.EventType))
	logString := errorPrefix + pipelineKeyPrefix + eventTypePrefix
	if event.Message != "" {
		logString += fmt.Sprintf(" %s\n", event.Message)
	}
	if event.Error != nil && len(event.Error.Exceptions) > 0 {
		logString += "trace for most recent exception: \n"
		for i := 0; i < len(event.Error.Exceptions); i++ {
			logString += fmt.Sprintf("%s\n", event.Error.Exceptions[i].Message)
		}
	}
	if logString != errorPrefix {
		log.Print(logString)
	}
}

func (r *pipelineRunner) logErrorEvent(ctx context.Context, pipelineId string, updateId string) error {

	w := r.bundle.WorkspaceClient()
	res, err := w.Pipelines.Impl().ListPipelineEvents(ctx, pipelines.ListPipelineEvents{
		Filter:     `level='ERROR'`,
		MaxResults: 100,
		PipelineId: pipelineId,
	})
	if err != nil {
		return err
	}
	// Note: For a 100 percent correct solution we should use the pagination token to find
	// a last event which took place for updateId incase it's not present in the first 100 events.
	// However the probablity of the error event not being present in the last 100 events
	// for the pipeline are should be very close 0, and this would not be worth the additional
	// complexity and latency cost for that extremely rare edge case
	updateEvents := filterEventsByUpdateId(res.Events, updateId)
	for i := len(updateEvents) - 1; i >= 0; i-- {
		r.logEvent(updateEvents[i])
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

func (r *pipelineRunner) Run(ctx context.Context, opts *Options) error {
	var prefix = fmt.Sprintf("[INFO] [%s]", r.Key())
	var pipelineID = r.pipeline.ID

	w := r.bundle.WorkspaceClient()
	_, err := w.Pipelines.GetByPipelineId(ctx, pipelineID)
	if err != nil {
		log.Printf("[WARN] Cannot get pipeline: %s", err)
		return err
	}

	req, err := opts.Pipeline.toPayload(pipelineID)
	if err != nil {
		return err
	}

	res, err := w.Pipelines.StartUpdate(ctx, *req)
	if err != nil {
		return err
	}

	updateID := res.UpdateId

	// Log the pipeline update URL as soon as it is available.
	updateUrl := fmt.Sprintf("%s/#joblist/pipelines/%s/updates/%s", w.Config.Host, pipelineID, updateID)
	log.Printf("%s Update available at %s", prefix, updateUrl)

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
			log.Printf("%s Update status: %s", prefix, state)
			prevState = &state
		}

		if state == pipelines.UpdateInfoStateCanceled {
			log.Printf("%s Update was cancelled!", prefix)
			return fmt.Errorf("update cancelled")
		}
		if state == pipelines.UpdateInfoStateFailed {
			log.Printf("%s Update has failed!", prefix)
			err := r.logErrorEvent(ctx, pipelineID, updateID)
			if err != nil {
				return err
			}
			return fmt.Errorf("update failed")
		}
		if state == pipelines.UpdateInfoStateCompleted {
			log.Printf("%s Update has completed successfully!", prefix)
			return nil
		}

		time.Sleep(time.Second)
	}
}
