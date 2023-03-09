package run

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"net/url"

	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/fatih/color"
	flag "github.com/spf13/pflag"
)

// TODO: Use a sdk implementation of this API once it's incorporated in the openapi
// spec. https://databricks.atlassian.net/browse/DECO-573
type pipelineEventErrorException struct {
	ClassName string `json:"class_name"`
	Message   string `json:"message"`
}

type pipelineEventError struct {
	Exceptions []pipelineEventErrorException `json:"exceptions"`
}

type pipelineEventOrigin struct {
	UpdateId string `json:"update_id"`
}

type pipelineEvent struct {
	Error   *pipelineEventError `json:"error"`
	Message string              `json:"message"`
	Origin  pipelineEventOrigin `json:"origin"`
}

type pipelineEventsResponse struct {
	Events []pipelineEvent `json:"events"`
}

func (r *pipelineRunner) logErrorEvent(ctx context.Context, pipelineId string, updateId string) error {
	apiClient, err := client.New(r.bundle.WorkspaceClient().Config)
	if err != nil {
		return err
	}
	filter := url.QueryEscape(`level='ERROR'`)
	apiPath := fmt.Sprintf("/api/2.0/pipelines/%s/events?filter=%s&max_results=100", pipelineId, filter)
	res := pipelineEventsResponse{}
	err = apiClient.Do(ctx, http.MethodGet, apiPath, nil, &res)
	if err != nil {
		return err
	}
	if len(res.Events) == 0 {
		return nil
	}
	var latestEvent *pipelineEvent
	// Note: For a 100 percent correct solution we should use the pagination token to find
	// a last event which took place for updateId incase it's not present in the first 100 events.
	// However the changes of the error event not being present in the last 100 events
	// for the pipeline are should be very close 0, and this would not be worth the additional
	// complexity and latency cost for that extremely rare edge case
	for i := 0; i < len(res.Events); i++ {
		if res.Events[i].Origin.UpdateId == updateId {
			latestEvent = &res.Events[i]
			break
		}
	}
	if latestEvent == nil {
		return nil
	}
	red := color.New(color.FgRed).SprintFunc()
	errorPrefix := fmt.Sprintf("%s [%s]", red("[ERROR]"), r.Key())
	logString := errorPrefix
	if latestEvent.Message != "" {
		logString += fmt.Sprintf(" %s\n", latestEvent.Message)
	}
	if latestEvent.Error != nil && len(latestEvent.Error.Exceptions) > 0 {
		logString += "trace for most recent exception: \n"
		for i := 0; i < len(latestEvent.Error.Exceptions); i++ {
			logString += fmt.Sprintf("%s\n", latestEvent.Error.Exceptions[i].Message)
		}
	}
	if logString != errorPrefix {
		log.Print(logString)
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
