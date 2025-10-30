// Copied from cmd/bundle/run.go and adapted for pipelines use.
// Consider if changes made here should be made to the bundle counterpart as well.
package pipelines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	bundleresources "github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	bundlerunoutput "github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/cmd/bundle/utils"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdgroup"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

type PipelineUpdateData struct {
	PipelineId    string
	Update        pipelines.UpdateInfo
	LastEventTime string
}

type ProgressEventWithDuration struct {
	Event    pipelines.PipelineEvent
	Duration string
	Phase    string
}

type ProgressEventsData struct {
	ProgressEvents []ProgressEventWithDuration
}

// phaseFromUpdateProgress extracts the phase name from an event message by checking if it contains any of the UpdateInfoState values
// Example: "Update 6fc8a8 is WAITING_FOR_RESOURCES." -> "WAITING_FOR_RESOURCES"
func phaseFromUpdateProgress(eventMessage string) (string, error) {
	var updateInfoState pipelines.UpdateInfoState
	updateInfoStates := updateInfoState.Values()

	for _, state := range updateInfoStates {
		if strings.Contains(eventMessage, string(state)) {
			return string(state), nil
		}
	}

	return "", fmt.Errorf("no phase found in message: %s", eventMessage)
}

// readableDuration returns a readable duration string for a given duration.
func readableDuration(diff time.Duration) (string, error) {
	if diff < 0 {
		return "", fmt.Errorf("duration cannot be negative: %v", diff)
	}

	if diff < time.Second {
		milliseconds := int(diff.Milliseconds())
		return fmt.Sprintf("%dms", milliseconds), nil
	}

	if diff < time.Minute {
		return fmt.Sprintf("%.1fs", diff.Seconds()), nil
	}

	if diff < time.Hour {
		minutes := int(diff.Minutes())
		seconds := int(diff.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds), nil
	}

	hours := int(diff.Hours())
	minutes := int(diff.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes), nil
}

// eventTimeDifference returns the time difference between two events.
func eventTimeDifference(earlierEvent, laterEvent pipelines.PipelineEvent) (time.Duration, error) {
	earlierTime, err := time.Parse(time.RFC3339Nano, earlierEvent.Timestamp)
	if err != nil {
		return 0, err
	}
	laterTime, err := time.Parse(time.RFC3339Nano, laterEvent.Timestamp)
	if err != nil {
		return 0, err
	}

	timeDifference := laterTime.Sub(earlierTime)
	if timeDifference < 0 {
		return 0, errors.New("second event timestamp must be after first event timestamp")
	}
	return timeDifference, nil
}

// enrichEvents adds duration information and phase name to each progress event.
// Expects that the events are already sorted by timestamp in ascending order.
// For the last event, duration is calculated using endTime.
func enrichEvents(events []pipelines.PipelineEvent, endTime string) ([]ProgressEventWithDuration, error) {
	var progressEventsWithDuration []ProgressEventWithDuration
	for j := range events {
		var nextEvent pipelines.PipelineEvent
		event := events[j]
		if j == len(events)-1 {
			nextEvent = pipelines.PipelineEvent{Timestamp: endTime}
		} else {
			nextEvent = events[j+1]
		}
		timeDifference, err := eventTimeDifference(event, nextEvent)
		if err != nil {
			return nil, err
		}
		readableDuration, err := readableDuration(timeDifference)
		if err != nil {
			return nil, err
		}
		phase, err := phaseFromUpdateProgress(event.Message)
		if err != nil {
			return nil, err
		}
		progressEventsWithDuration = append(progressEventsWithDuration, ProgressEventWithDuration{
			Event:    event,
			Duration: readableDuration,
			Phase:    phase,
		})
	}

	return progressEventsWithDuration, nil
}

// displayProgressEventsDurations displays the progress events with duration and phase name.
// Omits displaying the time of the last event.
func displayProgressEventsDurations(ctx context.Context, events []pipelines.PipelineEvent) error {
	if len(events) <= 1 {
		return nil
	}
	progressEvents, err := enrichEvents(events[:len(events)-1], getLastEventTime(events))
	if err != nil {
		return fmt.Errorf("failed to enrich progress events: %w", err)
	}

	data := ProgressEventsData{
		ProgressEvents: progressEvents,
	}

	return cmdio.RenderWithTemplate(ctx, data, "", progressEventsTemplate)
}

// fetchAndDisplayPipelineUpdate fetches the update and the update's associated update_progress events' durations.
func fetchAndDisplayPipelineUpdate(ctx context.Context, w *databricks.WorkspaceClient, pipelineId, updateId string) error {
	if pipelineId == "" {
		return errors.New("no pipeline ID provided")
	}
	if updateId == "" {
		return errors.New("no update ID provided")
	}

	getUpdateResponse, err := w.Pipelines.GetUpdate(ctx, pipelines.GetUpdateRequest{
		PipelineId: pipelineId,
		UpdateId:   updateId,
	})
	if err != nil {
		return err
	}

	if getUpdateResponse.Update == nil {
		return fmt.Errorf("no update found with id %s for pipeline %s", updateId, pipelineId)
	}

	latestUpdate := *getUpdateResponse.Update

	params := &PipelineEventsQueryParams{
		Filter:  fmt.Sprintf("update_id='%s' AND event_type='update_progress'", updateId),
		OrderBy: "timestamp asc",
	}

	events, err := fetchAllPipelineEvents(ctx, w, pipelineId, params)
	if err != nil {
		return err
	}

	err = displayPipelineUpdate(ctx, latestUpdate, pipelineId, events)
	if err != nil {
		return err
	}

	err = displayProgressEventsDurations(ctx, events)
	if err != nil {
		return err
	}

	return nil
}

// getLastEventTime returns the timestamp of the last progress event.
// Expects that the events are already sorted by timestamp in ascending order.
func getLastEventTime(events []pipelines.PipelineEvent) string {
	if len(events) == 0 {
		return ""
	}
	lastEvent := events[len(events)-1]
	parsedTime, err := time.Parse(time.RFC3339Nano, lastEvent.Timestamp)
	if err != nil {
		return ""
	}
	return parsedTime.Format(time.RFC3339Nano)
}

func displayPipelineUpdate(ctx context.Context, update pipelines.UpdateInfo, pipelineId string, events []pipelines.PipelineEvent) error {
	data := PipelineUpdateData{
		PipelineId:    pipelineId,
		Update:        update,
		LastEventTime: getLastEventTime(events),
	}

	return cmdio.RenderWithTemplate(ctx, data, "", pipelineUpdateTemplate)
}

func runCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run [flags] [KEY]",
		Short: "Run a pipeline",
		Long: `Run the pipeline identified by KEY.
KEY is the unique name of the pipeline to run, as defined in its YAML file.
If there is only one pipeline in the project, KEY is optional and the pipeline will be auto-selected.
Refreshes all tables in the pipeline unless otherwise specified.`,
	}

	var refresh []string
	var fullRefreshAll bool
	var fullRefresh []string

	pipelineGroup := cmdgroup.NewFlagGroup("Pipeline Run")
	pipelineGroup.FlagSet().StringSliceVar(&refresh, "refresh", nil, "List of tables to run.")
	pipelineGroup.FlagSet().BoolVar(&fullRefreshAll, "full-refresh-all", false, "Perform a full graph reset and recompute.")
	pipelineGroup.FlagSet().StringSliceVar(&fullRefresh, "full-refresh", nil, "List of tables to reset and recompute.")

	wrappedCmd := cmdgroup.NewCommandWithGroupFlag(cmd)
	wrappedCmd.AddFlagGroup(pipelineGroup)

	var noWait bool
	var restart bool
	cmd.Flags().BoolVar(&noWait, "no-wait", false, "Don't wait for the run to complete.")
	cmd.Flags().BoolVar(&restart, "restart", false, "Restart the run if it is already running.")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var key string
		b, err := utils.ProcessBundle(cmd, &utils.ProcessOptions{
			PostInitFunc: func(ctx context.Context, b *bundle.Bundle) error {
				var err error
				key, _, err = resolveRunArgument(ctx, b, args)
				return err
			},
			ErrorOnEmptyState: true,
		})
		if err != nil {
			return err
		}
		ctx := cmd.Context()

		runner, err := keyToRunner(b, key)
		if err != nil {
			return err
		}

		runOptions := run.Options{
			Pipeline: run.PipelineOptions{
				Refresh:        refresh,
				FullRefreshAll: fullRefreshAll,
				FullRefresh:    fullRefresh,
			},
			NoWait: noWait,
		}

		var runOutput bundlerunoutput.RunOutput
		if restart {
			runOutput, err = runner.Restart(ctx, &runOptions)
		} else {
			runOutput, err = runner.Run(ctx, &runOptions)
		}
		if err != nil {
			return err
		}

		if runOutput != nil {
			switch root.OutputType(cmd) {
			case flags.OutputText:
				resultString, err := runOutput.String()
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write([]byte(resultString))
				if err != nil {
					return err
				}
			case flags.OutputJSON:
				b, err := json.MarshalIndent(runOutput, "", "  ")
				if err != nil {
					return err
				}
				_, err = cmd.OutOrStdout().Write(b)
				if err != nil {
					return err
				}
				_, _ = cmd.OutOrStdout().Write([]byte{'\n'})
			default:
				return fmt.Errorf("unknown output type %s", root.OutputType(cmd))
			}
		}

		ref, err := bundleresources.Lookup(b, key, run.IsRunnable)
		if err != nil {
			return err
		}
		// Only displays the following pipeline run summary if the pipeline completes successfully,
		// as runner.Run() returns an error if the pipeline doesn't complete successfully.
		if ref.Description.SingularName == "pipeline" && runOutput != nil {
			if pipelineOutput, ok := runOutput.(*bundlerunoutput.PipelineOutput); ok && pipelineOutput.UpdateId != "" {
				w := b.WorkspaceClient()
				err = fetchAndDisplayPipelineUpdate(ctx, w, ref.Resource.(*resources.Pipeline).ID, pipelineOutput.UpdateId)
				if err != nil {
					return fmt.Errorf("failed to fetch and display pipeline update: %w", err)
				}
			}
		}
		return nil
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := root.MustConfigureBundle(cmd)
		if logdiag.HasError(cmd.Context()) {
			return nil, cobra.ShellCompDirectiveError
		}

		// No completion in the context of a bundle.
		// Source and destination paths are taken from bundle configuration.
		if b == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		if len(args) == 0 {
			completions := bundleresources.Completions(b, run.IsRunnable)
			return maps.Keys(completions), cobra.ShellCompDirectiveNoFileComp
		} else {
			// If we know the resource to run, we can complete additional positional arguments.
			runner, err := keyToRunner(b, args[0])
			if err != nil {
				return nil, cobra.ShellCompDirectiveError
			}
			return runner.CompleteArgs(args[1:], toComplete)
		}
	}

	return cmd
}
