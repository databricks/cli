package pipelines

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

// buildFieldFilter creates a SQL filter condition for a field with multiple possible values.
// It generates either "field = 'value'" for single values or "field in ('value1', 'value2')" for multiple values.
// This is used to build filter strings for the Databricks Pipelines API.
func buildFieldFilter(field string, values []string) string {
	if len(values) == 0 {
		return ""
	}
	if len(values) == 1 {
		return fmt.Sprintf("%s = '%s'", field, values[0])
	}
	valuesWithQuotes := make([]string, len(values))
	for i, value := range values {
		valuesWithQuotes[i] = fmt.Sprintf("'%s'", value)
	}
	return fmt.Sprintf("%s in (%s)", field, strings.Join(valuesWithQuotes, ", "))
}

func logsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [flags] PIPELINE_ID",
		Short: "Retrieve events for a pipeline",
		Long: `Retrieve events for the pipeline identified by PIPELINE_ID.

Examples:
  # Get all events for a pipeline and specific update ID
    pipelines logs pipeline-123 --update-id update-123

  # Get multiple log levels (ERROR and METRIC) for a specific event type (update_progress)
    pipelines logs pipeline-123 --level ERROR --level METRIC --event-type update_progress`,
	}

	var updateId string
	var levels []string
	var eventTypes []string
	var maxResults int

	cmd.Flags().StringVar(&updateId, "update-id", "", "Filter events by update ID.")
	cmd.Flags().StringSliceVar(&levels, "level", nil, "Filter events by log level (INFO, WARN, ERROR, METRIC, DEBUG). Can be specified multiple times.")
	cmd.Flags().StringSliceVar(&eventTypes, "event-type", nil, "Filter events by event type. Can be specified multiple times.")
	cmd.Flags().IntVar(&maxResults, "max-results", 100, "Max number of entries to return in a single page (<= 1000).")

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if len(args) == 0 {
			return errors.New("Provide a PIPELINE_ID.")
		}

		if len(args) > 1 {
			return fmt.Errorf("Expected one PIPELINE_ID, got %d.", len(args))
		}
		w := cmdctx.WorkspaceClient(ctx)

		pipelineId := args[0]

		var filterParts []string
		if updateId != "" {
			filterParts = append(filterParts, fmt.Sprintf("update_id = '%s'", updateId))
		}

		if levelFilter := buildFieldFilter("level", levels); levelFilter != "" {
			filterParts = append(filterParts, levelFilter)
		}

		if typeFilter := buildFieldFilter("event_type", eventTypes); typeFilter != "" {
			filterParts = append(filterParts, typeFilter)
		}

		var filter string
		if len(filterParts) > 0 {
			filter = strings.Join(filterParts, " AND ")
		}

		req := pipelines.ListPipelineEventsRequest{
			PipelineId: pipelineId,
			Filter:     filter,
			MaxResults: maxResults,
		}

		response := w.Pipelines.ListPipelineEvents(ctx, req)
		return cmdio.RenderIterator(ctx, response)
	}

	return cmd
}
