package pipelines

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
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

	filterGroup := cmdgroup.NewFlagGroup("Event Filter")
	filterGroup.FlagSet().StringVar(&updateId, "update-id", "", "Filter events by update ID.")
	filterGroup.FlagSet().StringSliceVar(&levels, "level", nil, "Filter events by list of log levels (INFO, WARN, ERROR, METRICS). ")
	filterGroup.FlagSet().StringSliceVar(&eventTypes, "event-type", nil, "Filter events by list of event types.")
	filterGroup.FlagSet().IntVar(&maxResults, "max-results", 100, "Max number of events to return.")

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

		req := pipelines.ListPipelineEventsRequest{
			PipelineId: pipelineId,
			Filter:     filter,
		}

		// TODO: change function to one that is unpaginated, so it supports the OrderBy parameter.
		// By default, events are returned in descending order by timestamp.
		iterator := w.Pipelines.ListPipelineEvents(ctx, req)

		limitedEvents, err := listing.ToSliceN(ctx, iterator, maxResults)
		if err != nil {
			return err
		}

		return cmdio.Render(ctx, limitedEvents)
	}

	return cmd
}
