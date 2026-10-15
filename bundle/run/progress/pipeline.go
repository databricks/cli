package progress

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/utils"
	sdkpipelines "github.com/databricks/sdk-go/pipelines/v2"
)

// The dlt backend computes events for pipeline runs which are accessible through
// the 2.0/pipelines/{pipeline_id}/events API
//
// There are 4 levels for these events: ("ERROR", "WARN", "INFO", "METRICS")
//
// Here's short introduction to a few important events we display on the console:
//
// 1. `update_progress`: A state transition occurred for the entire pipeline update
// 2. `flow_progress`: A state transition occurred for a single flow in the pipeine
type ProgressEvent pipelines.PipelineEvent

func (event *ProgressEvent) String() string {
	result := strings.Builder{}
	result.WriteString(utils.Deref(event.Timestamp) + " ")

	// Print event type with some padding to make output more pretty
	result.WriteString(fmt.Sprintf("%-15s", utils.Deref(event.EventType)) + " ")

	result.WriteString(string(event.Level) + " ")
	fmt.Fprintf(&result, `"%s"`, utils.Deref(event.Message))

	// construct error string if level=`Error`
	if event.Level == pipelines.EventLevel_Error && event.Error != nil {
		for _, exception := range event.Error.Exceptions {
			result.WriteString("\n" + utils.Deref(exception.Message))
		}
	}
	return result.String()
}

type UpdateTracker struct {
	UpdateId             string
	PipelineId           string
	LatestEventTimestamp string
	w                    *sdkclients.Clients
}

func NewUpdateTracker(pipelineId, updateId string, w *sdkclients.Clients) *UpdateTracker {
	return &UpdateTracker{
		w:                    w,
		PipelineId:           pipelineId,
		UpdateId:             updateId,
		LatestEventTimestamp: "",
	}
}

// To keep the logic simple we do not use pagination. This means that if there are
// more than 100 new events since the last query then we will miss out on progress events.
//
// This is fine because:
// 1. This should happen fairly rarely if ever
// 2. There is no expectation of the console progress logs being a complete representation
//
// # If a user needs the complete logs, they can always visit the run URL
//
// NOTE: Incase we want inplace logging, then we will need to implement pagination
func (l *UpdateTracker) Events(ctx context.Context) ([]ProgressEvent, error) {
	// create filter to fetch only new events
	filter := fmt.Sprintf(`update_id = '%s'`, l.UpdateId)
	if l.LatestEventTimestamp != "" {
		filter = filter + fmt.Sprintf(" AND timestamp > '%s'", l.LatestEventTimestamp)
	}

	// we only check the most recent 100 events for progress
	events, err := ListEvents(ctx, l.w, sdkpipelines.ListPipelineEventsRequest{
		PipelineId: new(l.PipelineId),
		MaxResults: new(100),
		Filter:     new(filter),
	})
	if err != nil {
		return nil, err
	}

	var result []ProgressEvent
	// Return events in chronological order.
	for _, event := range slices.Backward(events) {
		// filter to only include update_progress and flow_progress events
		eventType := utils.Deref(event.EventType)
		if eventType == "flow_progress" || eventType == "update_progress" {
			result = append(result, ProgressEvent(event))
		}
	}

	// update latest event timestamp for next time
	if len(result) > 0 {
		l.LatestEventTimestamp = utils.Deref(result[len(result)-1].Timestamp)
	}

	return result, nil
}

// ListEvents returns all pipeline events matching req, most recent first.
func ListEvents(ctx context.Context, w *sdkclients.Clients, req sdkpipelines.ListPipelineEventsRequest) ([]pipelines.PipelineEvent, error) {
	var events []pipelines.PipelineEvent
	for event, err := range w.Pipelines.EventsIter(ctx, req) {
		if err != nil {
			return nil, err
		}
		e, err := pipelines.PipelineEventFromSDK(event)
		if err != nil {
			return nil, err
		}
		events = append(events, *e)
	}
	return events, nil
}
