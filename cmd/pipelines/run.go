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
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

type PipelineUpdateData struct {
	PipelineId    string
	Update        pipelines.UpdateInfo
	LastEventTime string
}

// fetchAndDisplayPipelineUpdate fetches the latest update for a pipeline and displays information about it.
func fetchAndDisplayPipelineUpdate(ctx context.Context, bundle *bundle.Bundle, ref bundleresources.Reference, updateId string) error {
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
	return parsedTime.Format("2006-01-02T15:04:05Z")
}

func displayPipelineUpdate(ctx context.Context, update pipelines.UpdateInfo, pipelineID string, events []pipelines.PipelineEvent) error {
	data := PipelineUpdateData{
		PipelineId:    pipelineID,
		Update:        update,
		LastEventTime: getLastEventTime(events),
	}

	return cmdio.RenderWithTemplate(ctx, data, "", pipelineUpdateTemplate)


{{- if .ProgressEvents }}
{{- printf "%-50s %-7s\n" "Run Phase" "Duration" }}
{{- range $index, $event := .ProgressEvents }}
{{- if ne $index (sub (len $.ProgressEvents) 1) }}
{{- printf "%-50s %-7s\n" $event.Event.Message $event.Duration }}
{{- end }}
{{- end }}
{{- end }}

`

// PipelineUpdateData holds the data for rendering a single pipeline update
type PipelineUpdateData struct {
	PipelineId          string
	Update              pipelines.UpdateInfo
	ProgressEvents      []ProgressEventWithDuration
	RefreshSelectionStr string
	LastEventTime       string
	LatestErrorEvent    *pipelines.PipelineEvent
>>>>>>> ebb295ce3 (working default case)
}

// ProgressEventWithDuration adds duration information to a progress event
type ProgressEventWithDuration struct {
	Event      pipelines.PipelineEvent
	Duration   string
	ParsedTime time.Time
}

// getRefreshSelectionString returns a formatted string describing the refresh selection
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

// getLastEventTime returns the timestamp of the last progress event
func getLastEventTime(events []ProgressEventWithDuration) string {
	if len(events) == 0 {
		return ""
	}
	return events[len(events)-1].ParsedTime.Format("2006-01-02T15:04:05Z")
}

// getLatestErrorEvent finds the most recent error event from progress events
func getLatestErrorEvent(events []ProgressEventWithDuration) *pipelines.PipelineEvent {
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i].Event
		if event.Level == pipelines.EventLevelError {
			return &event
		}
	}
	return nil
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

		var runOutput output.RunOutput
		var runOutput bundlerunoutput.RunOutput
		if restart {
			runOutput, err = runner.Restart(ctx, &runOptions)
			runOutput, err = runner.Restart(ctx, &runOptions)
		} else {
			runOutput, err = runner.Run(ctx, &runOptions)
			runOutput, err = runner.Run(ctx, &runOptions)
		}
		if err != nil {
			return err
		}

		if runOutput != nil {
		if runOutput != nil {
			switch root.OutputType(cmd) {
			case flags.OutputText:
				resultString, err := runOutput.String()
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
			if pipelineOutput, ok := runOutput.(*output.PipelineOutput); ok && pipelineOutput.UpdateId != "" {
				err = fetchAndDisplayPipelineUpdate(ctx, b, ref, pipelineOutput.UpdateId)
				if err != nil {
					return err
				}
			}
		}
		ref, err := bundleresources.Lookup(b, key, run.IsRunnable)
		if err != nil {
			return err
		}
		if ref.Description.SingularName == "pipeline" && runOutput != nil {
			if pipelineOutput, ok := runOutput.(*bundlerunoutput.PipelineOutput); ok && pipelineOutput.UpdateId != "" {
				err = fetchAndDisplayPipelineUpdate(ctx, b, ref, pipelineOutput.UpdateId)
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

func fetchUpdateProgressEventsForUpdate(ctx context.Context, bundle *bundle.Bundle, pipelineId, updateId string) ([]pipelines.PipelineEvent, error) {
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
			return nil, fmt.Errorf("failed to get next event: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}

func calculateProgressEventsForUpdate(ctx context.Context, bundle *bundle.Bundle, update pipelines.UpdateInfo) ([]ProgressEventWithDuration, error) {
	events, err := fetchUpdateProgressEventsForUpdate(ctx, bundle, update.PipelineId, update.UpdateId)
	if err != nil {
		log.Warnf(ctx, "Failed to fetch events for update %s: %v", update.UpdateId, err)
		events = []pipelines.PipelineEvent{} // Use empty slice on error
	}

	var progressEventsWithDuration []ProgressEventWithDuration
	for j := len(events) - 1; j >= 0; j-- {
		event := events[j]
		duration := ""
		if j > 0 {
			currTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
			if err != nil {
				return nil, err
			}
			prevTime, err := time.Parse(time.RFC3339Nano, events[j-1].Timestamp)
			if err != nil {
				return nil, err
			}

			diff := prevTime.Sub(currTime)

			if diff > 0 {
				if diff < time.Minute {
					duration = fmt.Sprintf("%.1fs", diff.Seconds())
				} else if diff < time.Hour {
					minutes := int(diff.Minutes())
					seconds := int(diff.Seconds()) % 60
					duration = fmt.Sprintf("%dm %ds", minutes, seconds)
				} else {
					hours := int(diff.Hours())
					minutes := int(diff.Minutes()) % 60
					duration = fmt.Sprintf("%dh %dm", hours, minutes)
				}
			} else {
				duration = "0s"
			}
		}

		parsedTime, err := time.Parse(time.RFC3339Nano, event.Timestamp)
		if err != nil {
			return nil, err
		}

		progressEventsWithDuration = append(progressEventsWithDuration, ProgressEventWithDuration{
			Event:      event,
			Duration:   duration,
			ParsedTime: parsedTime,
		})
	}

	return progressEventsWithDuration, nil
}

func fetchAndDisplayPipelineUpdate(ctx context.Context, bundle *bundle.Bundle, ref bundleresources.Reference, updateId string) error {
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
		return fmt.Errorf("failed to fetch update %s: %w", updateId, err)
	}

	if getUpdateResponse.Update == nil {
		return fmt.Errorf("no update found with id %s", updateId)
	}

	latestUpdate := *getUpdateResponse.Update

	progressEvents, err := calculateProgressEventsForUpdate(ctx, bundle, latestUpdate)
	if err != nil {
		return fmt.Errorf("failed to calculate progress events: %w", err)
	}

	data := PipelineUpdateData{
		PipelineId:          pipelineID,
		Update:              latestUpdate,
		ProgressEvents:      progressEvents,
		RefreshSelectionStr: getRefreshSelectionString(latestUpdate),
		LastEventTime:       getLastEventTime(progressEvents),
		LatestErrorEvent:    getLatestErrorEvent(progressEvents),
	}

	return cmdio.RenderWithTemplate(ctx, data, "", pipelineUpdateTemplate)
}
