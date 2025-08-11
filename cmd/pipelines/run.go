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
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	bundleresources "github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	bundlerunoutput "github.com/databricks/cli/bundle/run/output"
	"github.com/databricks/cli/bundle/statemgmt"
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

// Extracts the last word from an event message and removes the last character (a period)
// Example: "Update 6fc8a8 is WAITING_FOR_RESOURCES." -> "WAITING_FOR_RESOURCES"
func phaseFromUpdateProgress(eventMessage string) string {
	words := strings.Fields(eventMessage)
	if len(words) > 0 {
		phase := words[len(words)-1]
		if len(phase) > 0 {
			phase = phase[:len(phase)-1]
		}
		return phase
	}
	return ""
}

// Returns a readable duration string for a given duration.
func readableDuration(diff time.Duration) string {
	if diff < time.Second {
		milliseconds := int(diff.Milliseconds())
		return fmt.Sprintf("%dms", milliseconds)
	}

	if diff < time.Minute {
		return fmt.Sprintf("%.1fs", diff.Seconds())
	}

	if diff < time.Hour {
		minutes := int(diff.Minutes())
		seconds := int(diff.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}

	hours := int(diff.Hours())
	minutes := int(diff.Minutes()) % 60
	return fmt.Sprintf("%dh %dm", hours, minutes)
}

// Returns the time difference between two events.
func eventTimeDifference(earlierEvent, laterEvent pipelines.PipelineEvent) (time.Duration, error) {
	currTime, err := time.Parse(time.RFC3339Nano, earlierEvent.Timestamp)
	if err != nil {
		return 0, err
	}
	nextTime, err := time.Parse(time.RFC3339Nano, laterEvent.Timestamp)
	if err != nil {
		return 0, err
	}

	timeDifference := nextTime.Sub(currTime)
	if timeDifference < 0 {
		return 0, errors.New("second event timestamp must be after first event timestamp")
	}
	return timeDifference, nil
}

// Adds duration information and phase name to a progress event
// Expects that the events are already sorted by timestamp in ascending order.
func enrichEvents(events []pipelines.PipelineEvent) ([]ProgressEventWithDuration, error) {
	var progressEventsWithDuration []ProgressEventWithDuration
	for j := 0; j < len(events)-1; j++ {
		event := events[j]
		nextEvent := events[j+1]
		timeDifference, err := eventTimeDifference(event, nextEvent)
		if err != nil {
			return nil, err
		}
		progressEventsWithDuration = append(progressEventsWithDuration, ProgressEventWithDuration{
			Event:    event,
			Duration: readableDuration(timeDifference),
			Phase:    phaseFromUpdateProgress(event.Message),
		})
	}

	return progressEventsWithDuration, nil
}

func displayProgressEvents(ctx context.Context, events []pipelines.PipelineEvent) error {
	progressEvents, err := enrichEvents(events)
	if err != nil {
		return fmt.Errorf("failed to calculate progress events: %w", err)
	}

	data := ProgressEventsData{
		ProgressEvents: progressEvents,
	}

	return cmdio.RenderWithTemplate(ctx, data, "", progressEventsTemplate)
}

// Displays the update and the update's associated update_progress events' durations.
func fetchAndDisplayPipelineUpdate(ctx context.Context, w *databricks.WorkspaceClient, ref bundleresources.Reference, updateId string) error {
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
		return fmt.Errorf("no update found with id %s for pipeline %s", updateId, pipelineID)
	}

	latestUpdate := *getUpdateResponse.Update

	params := &PipelineEventsQueryParams{
		Filter:  fmt.Sprintf("update_id='%s' AND event_type='update_progress'", updateId),
		OrderBy: "timestamp asc",
	}

	events, err := fetchAllPipelineEvents(ctx, w, pipelineID, params)
	if err != nil {
		return err
	}

	if latestUpdate.State == pipelines.UpdateInfoStateCompleted {
		err = displayPipelineUpdate(ctx, latestUpdate, pipelineID, events)
		if err != nil {
			return err
		}
	}

	err = displayProgressEvents(ctx, events)
	if err != nil {
		return err
	}

	return nil
}

// Returns the timestamp of the last progress event.
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
	return parsedTime.Format("2006-01-02T15:04:05Z")
}

func displayPipelineUpdate(ctx context.Context, update pipelines.UpdateInfo, pipelineID string, events []pipelines.PipelineEvent) error {
	data := PipelineUpdateData{
		PipelineId:    pipelineID,
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
		ctx := logdiag.InitContext(cmd.Context())
		cmd.SetContext(ctx)

		b := utils.ConfigureBundleWithVariables(cmd)
		if b == nil || logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		phases.Initialize(ctx, b)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

		key, _, err := resolveRunArgument(ctx, b, args)
		if err != nil {
			return err
		}

		if !b.DirectDeployment {
			bundle.ApplySeqContext(ctx, b,
				terraform.Interpolate(),
				terraform.Write(),
			)
			if logdiag.HasError(ctx) {
				return root.ErrAlreadyPrinted
			}
		}

		bundle.ApplySeqContext(ctx, b,
			statemgmt.StatePull(),
			statemgmt.Load(statemgmt.ErrorOnEmptyState),
		)
		if logdiag.HasError(ctx) {
			return root.ErrAlreadyPrinted
		}

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
		if ref.Description.SingularName == "pipeline" && runOutput != nil {
			if pipelineOutput, ok := runOutput.(*bundlerunoutput.PipelineOutput); ok && pipelineOutput.UpdateId != "" {
				w := b.WorkspaceClient()
				err = fetchAndDisplayPipelineUpdate(ctx, w, ref, pipelineOutput.UpdateId)
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
