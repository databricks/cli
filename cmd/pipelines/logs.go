package pipelines

import (
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
)

// Finds the update with the most recent CreationTime from a list of updates.
func getMostRecentUpdateId(updates []pipelines.UpdateInfo) (string, error) {
	if len(updates) == 0 {
		return "", errors.New("no updates provided")
	}

	var mostRecentUpdate *pipelines.UpdateInfo
	var mostRecentTime int64 = 0

	for i := range updates {
		update := &updates[i]
		if update.CreationTime > mostRecentTime {
			mostRecentTime = update.CreationTime
			mostRecentUpdate = update
		}
	}

	if mostRecentUpdate == nil {
		return "", errors.New("no valid updates found")
	}

	return mostRecentUpdate.UpdateId, nil
}

// Creates a SQL filter condition for a field with multiple possible values,
// generating "field in ('value1')" for a single value or "field in ('value1', 'value2')" for multiple values.
func buildFieldFilter(field string, values []string) string {
	if len(values) == 0 {
		return ""
	}

	quotedValues := "'" + strings.Join(values, "', '") + "'"
	return fmt.Sprintf("%s in (%s)", field, quotedValues)
}

// Cconstructs a SQL filter string for pipeline events based on the provided parameters.
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
By default, show events for the pipeline's most recent update.

Example usage:
  1. pipelines logs my-pipeline --update-id update-1
  2. pipelines logs my-pipeline --level ERROR,METRICS --event-type update_progress`,
	}

	var updateId string
	var levels []string
	var eventTypes []string
	var number int

	filterGroup := cmdgroup.NewFlagGroup("Event Filter")
	filterGroup.FlagSet().StringVar(&updateId, "update-id", "", "Filter events by update ID. If not provided, uses the most recent update ID.")
	filterGroup.FlagSet().StringSliceVar(&levels, "level", nil, "Filter events by list of log levels (INFO, WARN, ERROR, METRICS). ")
	filterGroup.FlagSet().StringSliceVar(&eventTypes, "event-type", nil, "Filter events by list of event types.")
	filterGroup.FlagSet().IntVarP(&number, "number", "n", 0, "Number of events to return.")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(filterGroup)

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if len(args) == 0 {
			return errors.New("provide a PIPELINE_ID")
		}

		if len(args) > 1 {
			return fmt.Errorf("expected one PIPELINE_ID, got %d", len(args))
		}
		w := cmdctx.WorkspaceClient(ctx)

		pipelineId := args[0]

		if updateId == "" {
			allUpdates, err := fetchAllUpdates(ctx, w, pipelineId)
			if err != nil {
				return err
			}

			updateId, err = getMostRecentUpdateId(allUpdates)
			if err != nil {
				return err
			}
		}

		filter := buildPipelineEventFilter(updateId, levels, eventTypes)

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
			return err
		}

		return cmdio.Render(ctx, events)
	}

	return cmd
}
