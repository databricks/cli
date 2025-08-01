package output

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	bundleresources "github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"golang.org/x/exp/slices"
)

type PipelineUpdateData struct {
	PipelineId          string
	Update              pipelines.UpdateInfo
	RefreshSelectionStr string
	LastEventTime       string
}

const pipelineUpdateTemplate = `Update {{ .Update.UpdateId }} for pipeline {{- if .Update.Config }}{{ .Update.Config.Name }}{{ end }} {{- if .Update.Config }}{{ .Update.Config.Id }}{{ end }} completed successfully.
{{- if .Update.Cause }}
Cause: {{ .Update.Cause }}
{{- end }}
{{- if .Update.CreationTime }}
Creation Time: {{ .Update.CreationTime | pretty_UTC_date_from_millis }}
{{- end }}
{{- if .LastEventTime }}
End Time: {{ .LastEventTime }}
{{- end }}
{{- if or (and .Update.Config .Update.Config.Serverless) .Update.ClusterId }}
Compute: {{ if .Update.Config.Serverless }} serverless {{ else }}{{ .Update.ClusterId }}{{ end }}
{{- end }}
Refresh: {{ .RefreshSelectionStr }}
{{- if .Update.Config }}
{{- if .Update.Config.Channel }}
Channel: {{ .Update.Config.Channel }}
{{- end }}
{{- if .Update.Config.Continuous }}
Continuous: {{ .Update.Config.Continuous }}
{{- end }}
{{- if .Update.Config.Development }}
Development mode: {{ if .Update.Config.Development }}Dev{{ else }}Prod{{ end }}
{{- end }}
{{- if .Update.Config.Environment }}
Environment: {{ .Update.Config.Environment }}
{{- end }}
{{- if or .Update.Config.Catalog .Update.Config.Schema }}
Catalog & Schema: {{ .Update.Config.Catalog }}{{ if and .Update.Config.Catalog .Update.Config.Schema }}.{{ end }}{{ .Update.Config.Schema }}
{{- end }}
{{- end }}
`

func getRefreshSelectionString(update pipelines.UpdateInfo) string {
	if update.FullRefresh {
		return "full-refresh-all"
	}

	var parts []string
	if len(update.RefreshSelection) > 0 {
		parts = append(parts, fmt.Sprintf("refreshed [%s]", strings.Join(update.RefreshSelection, ", ")))
	}
	if len(update.FullRefreshSelection) > 0 {
		parts = append(parts, fmt.Sprintf("full-refreshed [%s]", strings.Join(update.FullRefreshSelection, ", ")))
	}

	if len(parts) > 0 {
		return strings.Join(parts, " | ")
	}

	return "default refresh-all"
}

func fetchUpdateProgressEventsForUpdateAscending(ctx context.Context, bundle *bundle.Bundle, pipelineId, updateId string) ([]pipelines.PipelineEvent, error) {
	w := bundle.WorkspaceClient()

	req := pipelines.ListPipelineEventsRequest{
		PipelineId: pipelineId,
		Filter:     fmt.Sprintf("update_id='%s' AND event_type='update_progress'", updateId),
		// OrderBy:    []string{"timestamp asc"}, TODO: Add this back in when the API is fixed
	}

	iterator := w.Pipelines.ListPipelineEvents(ctx, req)
	var events []pipelines.PipelineEvent

	for iterator.HasNext(ctx) {
		event, err := iterator.Next(ctx)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	slices.Reverse(events)

	return events, nil
}

func FetchAndDisplayPipelineUpdate(ctx context.Context, bundle *bundle.Bundle, ref bundleresources.Reference, updateId string) error {
	w := bundle.WorkspaceClient()

	pipelineResource := ref.Resource.(*resources.Pipeline)
	pipelineID := pipelineResource.ID
	if pipelineID == "" {
		return errors.New("unable to get pipeline ID from pipeline")
	}

	getUpdateResponse, err := w.Pipelines.GetUpdate(ctx, pipelines.GetUpdateRequest{
		PipelineId: pipelineID,
		UpdateId:   updateId,
	})
	if err != nil {
		return err
	}

	if getUpdateResponse.Update == nil {
		return err
	}

	latestUpdate := *getUpdateResponse.Update

	if latestUpdate.State == pipelines.UpdateInfoStateCompleted {
		events, err := fetchUpdateProgressEventsForUpdateAscending(ctx, bundle, pipelineID, updateId)
		if err != nil {
			return err
		}

		err = displayPipelineUpdate(ctx, latestUpdate, pipelineID, events)
		if err != nil {
			return err
		}
	}

	return nil
}

// getLastEventTime returns the timestamp of the last progress event
func getLastEventTime(events []pipelines.PipelineEvent) string {
	if len(events) == 0 {
		return ""
	}
	lastEvent := events[len(events)-1]
	parsedTime, err := time.Parse(time.RFC3339Nano, lastEvent.Timestamp)
	if err != nil {
		return ""
	}
	return parsedTime.Format("2006-01-02T15:04:05Z")
}

// displayPipelineUpdate displays pipeline update information
func displayPipelineUpdate(ctx context.Context, update pipelines.UpdateInfo, pipelineID string, events []pipelines.PipelineEvent) error {
	data := PipelineUpdateData{
		PipelineId:          pipelineID,
		Update:              update,
		RefreshSelectionStr: getRefreshSelectionString(update),
		LastEventTime:       getLastEventTime(events),
	}

	return cmdio.RenderWithTemplate(ctx, data, "", pipelineUpdateTemplate)
}
