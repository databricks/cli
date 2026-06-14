package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

// getData is the payload printed by `air get`. The json-tagged fields form
// the machine-readable output; fields tagged `json:"-"` are shown only in the
// human-readable text view.
type getData struct {
	RunID           string  `json:"run_id"`
	Status          string  `json:"status"`
	StartedAt       *string `json:"started_at"`
	DurationSeconds *int64  `json:"duration_seconds"`
	AttemptNumber   int     `json:"attempt_number"`
	ExperimentName  *string `json:"experiment_name"`
	DashboardURL    string  `json:"dashboard_url"`
	MLflowURL       *string `json:"mlflow_url"`

	// Duration is the human-readable form of DurationSeconds, e.g. "12m 3s".
	Duration string `json:"-"`
	// Accelerators describes the run's GPUs, e.g. "8x H100".
	Accelerators string `json:"-"`
	// User is the run's creator. Text-only; JSON omits it, matching `air get --json`.
	User string `json:"-"`
	// Sweep replaces the single-run view for foreach runs. Text-only; JSON omits it.
	Sweep *sweepInfo `json:"-"`
}

// getTemplate is the text-mode layout. It reads from the JSON envelope, so
// every field is reached through ".Data". Optional rows are hidden when empty.
const getTemplate = `{{- if .Data.Sweep -}}
Sweep Run ID: {{.Data.RunID}}
Status:       {{.Data.Status}}
Total:        {{.Data.Sweep.Total}}
Completed:    {{.Data.Sweep.Completed}}
Succeeded:    {{.Data.Sweep.Succeeded}}
Failed:       {{.Data.Sweep.Failed}}
Active:       {{.Data.Sweep.Active}}
{{- if .Data.Sweep.Tasks}}

Sweep Tasks:
{{printf "  %-24s %-14s %-12s %s" "TASK" "RUN ID" "STATUS" "EXPERIMENT"}}
{{- range .Data.Sweep.Tasks}}
{{printf "  %-24s %-14s %-12s %s" .TaskKey .RunID .Status .Experiment}}
{{- end}}
{{- end}}
{{- else -}}
Run ID:       {{.Data.RunID}}
Status:       {{.Data.Status}}
{{- if .Data.StartedAt}}
Submitted:    {{.Data.StartedAt}}
{{- end}}
{{- if .Data.Duration}}
Duration:     {{.Data.Duration}}
{{- end}}
Retries:      {{.Data.AttemptNumber}}
{{- if .Data.ExperimentName}}
Experiment:   {{.Data.ExperimentName}}
{{- end}}
{{- if .Data.User}}
User:         {{.Data.User}}
{{- end}}
{{- if .Data.Accelerators}}
Accelerators: {{.Data.Accelerators}}
{{- end}}
{{- if .Data.MLflowURL}}
MLflow:       {{.Data.MLflowURL}}
{{- end}}
Dashboard:    {{.Data.DashboardURL}}
{{- end}}
`

func newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get JOB_RUN_ID",
		Args:  root.ExactArgs(1),
		Short: "Show details for a run",
		Annotations: map[string]string{
			"template": getTemplate,
		},
	}

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		runID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || runID <= 0 {
			return fmt.Errorf("invalid RUN_ID %q: must be a positive integer", args[0])
		}

		run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
		if err != nil {
			// The backend returns this when the run ID is unknown to the user.
			if errors.Is(err, apierr.ErrResourceDoesNotExist) {
				return fmt.Errorf("run %d not found: check the run ID and that it is a job run ID", runID)
			}
			return fmt.Errorf("failed to get status for run %d: %w", runID, err)
		}

		data := buildGetData(run)
		data.MLflowURL = mlflowURL(ctx, w, run)
		if task := findForEachTask(run); task != nil {
			data.Sweep = buildSweepInfo(ctx, w, task)
		}

		// Text mode shows the training-config YAML before the status, mirroring
		// `air get`. JSON output omits it, matching `air get --json`.
		if root.OutputType(cmd) == flags.OutputText {
			if path := yamlConfigPath(run); path != "" {
				printConfigYAML(ctx, w, path)
			}
		}
		return renderEnvelope(ctx, data)
	}

	return cmd
}

// buildGetData extracts the fields we display from a run.
func buildGetData(run *jobs.Run) getData {
	data := getData{
		RunID:           strconv.FormatInt(run.RunId, 10),
		Status:          runStatus(run.State),
		StartedAt:       startedAt(run),
		DurationSeconds: durationSeconds(run),
		AttemptNumber:   latestAttemptNumber(run),
		ExperimentName:  experimentName(run),
		DashboardURL:    run.RunPageUrl,
		Accelerators:    accelerators(run),
		User:            run.CreatorUserName,
	}
	if data.DurationSeconds != nil {
		data.Duration = formatDuration(*data.DurationSeconds)
	}
	return data
}

// yamlConfigPath returns the run's training-config YAML path, or "" if none.
func yamlConfigPath(run *jobs.Run) string {
	if len(run.Tasks) == 0 {
		return ""
	}
	task := run.Tasks[0].GenAiComputeTask
	if task == nil {
		return ""
	}
	return task.YamlParametersFilePath
}

// printConfigYAML downloads the run's training-config YAML and prints it. It is
// best-effort: a failure is surfaced as a warning but does not fail status.
func printConfigYAML(ctx context.Context, w *databricks.WorkspaceClient, path string) {
	r, err := w.Workspace.Download(ctx, path)
	if err != nil {
		log.Warnf(ctx, "air get: could not download training config %s: %v", path, err)
		return
	}
	defer r.Close()

	content, err := io.ReadAll(r)
	if err != nil {
		log.Warnf(ctx, "air get: could not read training config %s: %v", path, err)
		return
	}

	cmdio.LogString(ctx, "Training Configuration:")
	cmdio.LogString(ctx, string(content))
	cmdio.LogString(ctx, "")
}
