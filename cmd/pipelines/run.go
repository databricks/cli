// Copied from cmd/bundle/run.go and adapted for pipelines use.
// Consider if changes made here should be made to the bundle counterpart as well.
package pipelines

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/phases"
	bundleresources "github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/bundle/run/output"
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

const pipelineUpdateTemplate = `State: {{ .Update.State }}
{{- if .Update.Cause }}
Cause: {{ .Update.Cause }}
{{- end }}
Creation Time: {{ .Update.CreationTime | pretty_UTC_date_from_millis}}
{{- if .Update.ClusterId }}
Cluster ID: {{ .Update.ClusterId }}
{{- end }}
Full Refresh: {{ .Update.FullRefresh | bool }}
Validate Only: {{ .Update.ValidateOnly | bool }}
{{- if .Update.RefreshSelection }}
Refresh Selection: {{ join .Update.RefreshSelection ", " }}
{{- end }}
{{- if .Update.FullRefreshSelection }}
Full Refresh Selection: {{ join .Update.FullRefreshSelection ", " }}
{{- end }}
{{- if .Update.Config }}
Pipeline Spec:
  {{- if .Update.Config.Id }}
  ID: {{ .Update.Config.Id }}
  {{- end }}
  {{- if .Update.Config.Name }}
  Name: {{ .Update.Config.Name }}
  {{- end }}
  {{- if .Update.Config.Catalog }}
  Catalog: {{ .Update.Config.Catalog }}
  {{- end }}
  {{- if .Update.Config.Channel }}
  Channel: {{ .Update.Config.Channel }}
  {{- end }}
  {{- if .Update.Config.Continuous }}
  Continuous: {{ .Update.Config.Continuous }}
  {{- end }}
  {{- if .Update.Config.Development }}
  Development: {{ .Update.Config.Development }}
  {{- end }}
  {{- if .Update.Config.Environment }}
  Environment: {{ .Update.Config.Environment }}
  {{- end }}
  {{- if .Update.Config.Schema }}
  Schema: {{ .Update.Config.Schema }}
  {{- end }}
  {{- if .Update.Config.Serverless }}
  Serverless: {{ .Update.Config.Serverless }}
  {{- end }}
  {{- if .Update.Config.Storage }}
  Storage: {{ .Update.Config.Storage }}
  {{- end }}
  {{- if not (or .Update.Config.Id .Update.Config.Name .Update.Config.BudgetPolicyId .Update.Config.Catalog .Update.Config.Channel .Update.Config.Continuous .Update.Config.Deployment .Update.Config.Development .Update.Config.Environment .Update.Config.EventLog .Update.Config.Filters .Update.Config.GatewayDefinition .Update.Config.IngestionDefinition .Update.Config.Notifications .Update.Config.RestartWindow .Update.Config.RootPath .Update.Config.Schema .Update.Config.Serverless .Update.Config.Storage) }}
  Raw Config: {{ .Update.Config }}
  {{- end }}
{{- else }}
Pipeline Spec: (No config available)
{{- end }}
{{- if .ProgressEvents }}
Progress Events:
{{- range $index, $event := .ProgressEvents }}
  - {{ $event.Event.Timestamp }} {{ $event.Event.EventType }} {{ $event.Event.Level }} "{{ $event.Event.Message }}"
  {{- if ne $index 0 }}
    Duration: {{ $event.DurationSincePrevious }}
  {{- end }}
{{- end }}
{{- end }}

`

// PipelineUpdateData holds the data for rendering a single pipeline update
type PipelineUpdateData struct {
	PipelineId     string
	Update         pipelines.UpdateInfo
	ProgressEvents []ProgressEventWithDuration
}

// ProgressEventWithDuration adds duration information to a progress event
type ProgressEventWithDuration struct {
	Event                 pipelines.PipelineEvent
	DurationSincePrevious string
	ParsedTime            time.Time
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
			if pipelineOutput, ok := runOutput.(*output.PipelineOutput); ok && pipelineOutput.UpdateId != "" {
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
	for j, event := range events {
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
			Event:                 event,
			DurationSincePrevious: duration,
			ParsedTime:            parsedTime,
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
		PipelineId:     pipelineID,
		Update:         latestUpdate,
		ProgressEvents: progressEvents,
	}

	return cmdio.RenderWithTemplate(ctx, data, "", pipelineUpdateTemplate)
}
