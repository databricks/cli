package pipelines

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/spf13/cobra"
)

// buildFieldFilter creates a SQL filter condition for a field with multiple possible values,
// generating "field in ('value1')" for a single value or "field in ('value1', 'value2')" for multiple values.
func buildFieldFilter(field string, values []string) string {
	if len(values) == 0 {
		return ""
	}

	quotedValues := "'" + strings.Join(values, "', '") + "'"
	return fmt.Sprintf("%s in (%s)", field, quotedValues)
}

// buildPipelineEventFilter constructs a SQL filter string for pipeline events based on the provided parameters.
func buildPipelineEventFilter(updateId string, levels, eventTypes []string) string {
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

	if len(filterParts) > 0 {
		return strings.Join(filterParts, " AND ")
	}

	return ""
}

func logsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [flags] PIPELINE_ID",
		Short: "Retrieve events for a pipeline",
		Long: `Retrieve events for the pipeline identified by PIPELINE_ID, a unique identifier for the pipeline.

Example usage:
  1. pipelines logs my-pipeline --update-id update-1
  2. pipelines logs my-pipeline --level ERROR,METRICS --event-type update_progress`,
	}

	var updateId string
	var levels []string
	var eventTypes []string
	var maxResults int
	var reverse bool

	filterGroup := cmdgroup.NewFlagGroup("Event Filter")
	filterGroup.FlagSet().StringVar(&updateId, "update-id", "", "Filter events by update ID.")
	filterGroup.FlagSet().StringSliceVar(&levels, "level", nil, "Filter events by list of log levels (INFO, WARN, ERROR, METRICS). ")
	filterGroup.FlagSet().StringSliceVar(&eventTypes, "event-type", nil, "Filter events by list of event types.")
	filterGroup.FlagSet().IntVar(&maxResults, "max-results", 100, "Max number of events to return.")
	filterGroup.FlagSet().BoolVar(&reverse, "r", false, "Reverse the order of results. By default, events are returned in descending order by timestamp.")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(filterGroup)

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

		filter := buildPipelineEventFilter(updateId, levels, eventTypes)

		orderBy := "timestamp desc"
		if reverse {
			orderBy = "timestamp asc"
		}

		params := &PipelineEventsQueryParams{
			Filter:     filter,
			MaxResults: 1000, // Use maximum page size
			OrderBy:    orderBy,
		}

		events, err := fetchAllPipelineEvents(ctx, w, pipelineId, params)
		if err != nil {
			return err
		}

		if len(events) > maxResults {
			events = events[:maxResults]
		}

		return cmdio.Render(ctx, events)
	}

	return cmd
}
