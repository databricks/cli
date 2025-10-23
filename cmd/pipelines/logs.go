package pipelines

import (
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/logdiag"

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
func buildPipelineEventFilter(updateId string, levels, eventTypes []string, startTime, endTime string) string {
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

	if startTime != "" {
		filterParts = append(filterParts, fmt.Sprintf("timestamp >= '%s'", startTime))
	}

	if endTime != "" {
		filterParts = append(filterParts, fmt.Sprintf("timestamp <= '%s'", endTime))
	}

	return strings.Join(filterParts, " AND ")
}

func logsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [flags] [KEY]",
		Args:  root.MaximumNArgs(1),
		Short: "Retrieve events for a pipeline",
		Long: `Retrieve events for the pipeline identified by KEY.
KEY is the unique name of the pipeline, as defined in its YAML file.
By default, show the events of the pipeline's most recent update.

Example usage:
  1. pipelines logs pipeline-name --update-id update-1 -n 10
  2. pipelines logs pipeline-name --level ERROR,METRICS --event-type update_progress --start-time 2025-01-15T10:30:00Z`,
	}

	var updateId string
	var levels []string
	var eventTypes []string
	var number int
	var startTime string
	var endTime string

	filterGroup := cmdgroup.NewFlagGroup("Event Filter")
	filterGroup.FlagSet().StringVar(&updateId, "update-id", "", "Filter events by update ID. If not provided, uses the most recent update ID.")
	filterGroup.FlagSet().StringSliceVar(&levels, "level", nil, "Filter events by list of log levels (INFO, WARN, ERROR, METRICS). ")
	filterGroup.FlagSet().StringSliceVar(&eventTypes, "event-type", nil, "Filter events by list of event types.")
	filterGroup.FlagSet().IntVarP(&number, "number", "n", 0, "Number of events to return.")
	filterGroup.FlagSet().StringVar(&startTime, "start-time", "", "Filter for events that are after this start time (format: 2025-01-15T10:30:00Z)")
	filterGroup.FlagSet().StringVar(&endTime, "end-time", "", "Filter for events that are before this end time (format: 2025-01-15T10:30:00Z)")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(filterGroup)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := utils.ConfigureBundleWithVariables(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}
		ctx = cmd.Context()

		phases.Initialize(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		// Load the deployment state to get pipeline IDs from resource
		ctx = statemgmt.PullResourcesState(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		bundle.ApplySeqContext(ctx, b,
			statemgmt.Load(),
		)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		key, err := resolvePipelineArgument(ctx, b, args)
		if err != nil {
			return err
		}

		pipelineId, err := resolvePipelineIdFromKey(ctx, b, key)
		if err != nil {
			return err
		}

		w := b.WorkspaceClient()
		if updateId == "" {
			updateId, err = getMostRecentUpdateId(ctx, w, pipelineId)
			if err != nil {
				return fmt.Errorf("failed to get most recent update ID: %w", err)
			}
		}

		if startTime != "" {
			startTime, err = parseAndFormatTimestamp(startTime)
			if err != nil {
				return fmt.Errorf("invalid start time format: %w", err)
			}
		}

		if endTime != "" {
			endTime, err = parseAndFormatTimestamp(endTime)
			if err != nil {
				return fmt.Errorf("invalid end time format: %w", err)
			}
		}

		filter := buildPipelineEventFilter(updateId, levels, eventTypes, startTime, endTime)

		params := &PipelineEventsQueryParams{
			Filter:  filter,
			OrderBy: "timestamp desc",
		}

		// Only set MaxResults if the flag was provided, avoiding setting to the default value.
		if cmd.Flags().Changed("number") {
			params.MaxResults = number
		}

		events, err := fetchAllPipelineEvents(ctx, w, pipelineId, params)
		if err != nil {
			return fmt.Errorf("failed to fetch events for pipeline %s with update ID %s: %w", pipelineId, updateId, err)
		}

		return cmdio.Render(ctx, events)
	}

	return cmd
}
