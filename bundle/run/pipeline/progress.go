package pipeline

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

type ProgressEvent pipelines.PipelineEvent

func (event *ProgressEvent) String() string {
	result := strings.Builder{}

	result.WriteString(event.Timestamp + " ")
	result.WriteString(event.EventType + " ")

	// add name of the subject flow/pipeline
	switch event.EventType {
	case "flow_progress":
		result.WriteString(event.Origin.FlowName + " ")
	case "update_progress":
		result.WriteString(event.Origin.PipelineName + " ")
	}
	result.WriteString(event.Level.String() + " ")
	result.WriteString(event.Message)

	return result.String()
}

// TODO: Add inplace logging to pipelines. https://github.com/databricks/bricks/issues/280
type UpdateTracker struct {
	UpdateId             string
	PipelineId           string
	LatestEventTimestamp string
	w                    *databricks.WorkspaceClient
}

func NewUpdateTracker(pipelineId string, updateId string, w *databricks.WorkspaceClient) *UpdateTracker {
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
	filter := ""
	if l.LatestEventTimestamp != "" {
		filter = fmt.Sprintf(`timestamp > '%s'`, l.LatestEventTimestamp)
	}

	// we only check the most recent 100 events for progress
	response, err := l.w.Pipelines.Impl().ListPipelineEvents(ctx, pipelines.ListPipelineEvents{
		PipelineId: l.PipelineId,
		MaxResults: 100,
		Filter:     filter,
	})
	if err != nil {
		return nil, err
	}

	// filter out update_progress and flow_progress events
	result := make([]ProgressEvent, 0)
	for _, event := range response.Events {
		if event.Origin.UpdateId != l.UpdateId {
			continue
		}
		if event.EventType == "flow_progress" || event.EventType == "update_progress" {
			result = append(result, ProgressEvent(event))
		}
	}

	// update latest event timestamp for next time
	if len(result) > 0 {
		l.LatestEventTimestamp = result[0].Timestamp
	}
	return result, nil
}
