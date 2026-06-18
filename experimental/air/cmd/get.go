package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

// getData is the payload printed by `air get run`. The json-tagged fields form
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

	// The fields below are pre-rendered for the text table and excluded from
	// JSON (matching `air get run --json`). The table always shows every row,
	// with "N/A" for missing values, in the same order as the Python CLI. The
	// Run ID, Experiment, and MLflow Run cells carry terminal hyperlinks when
	// stdout is a terminal, so the URLs don't appear as bare text.
	RunIDDisplay        string `json:"-"`
	SubmittedDisplay    string `json:"-"`
	DurationDisplay     string `json:"-"`
	ExperimentDisplay   string `json:"-"`
	MLflowDisplay       string `json:"-"`
	UserDisplay         string `json:"-"`
	AcceleratorsDisplay string `json:"-"`
	// Sweep replaces the single-run view for foreach runs.
	Sweep *sweepInfo `json:"-"`
}

// getTemplate is the text-mode layout. It reads from the JSON envelope, so
// every field is reached through ".Data".
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
Run ID:       {{.Data.RunIDDisplay}}
Status:       {{.Data.Status}}
Submitted:    {{.Data.SubmittedDisplay}}
Retries:      {{.Data.AttemptNumber}}
Duration:     {{.Data.DurationDisplay}}
Experiment:   {{.Data.ExperimentDisplay}}
MLflow Run:   {{.Data.MLflowDisplay}}
User:         {{.Data.UserDisplay}}
Accelerators: {{.Data.AcceleratorsDisplay}}
{{- end}}
`

// newGetCommand is the `get` parent group. Subcommands name the resource to
// describe, e.g. `air get run JOB_RUN_ID`, mirroring the Python CLI.
func newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Show details for a specific resource",
	}
	cmd.AddCommand(newGetRunCommand())
	return cmd
}

func newGetRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run JOB_RUN_ID",
		Args:  root.ExactArgs(1),
		Short: "Show status, configuration, and timing details for a specific run",
		Annotations: map[string]string{
			"template": getTemplate,
		},
	}

	// Match Python: a client/auth failure is a JSON error envelope in -o json mode,
	// not a bare error. ErrAlreadyPrinted passes through (it was handled upstream).
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		err := root.MustWorkspaceClient(cmd, args)
		if err == nil || errors.Is(err, root.ErrAlreadyPrinted) {
			return err
		}
		return renderError(cmd.Context(), cmd, "INTERNAL_ERROR", "TRANSIENT", true, err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		runID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || runID <= 0 {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				fmt.Errorf("invalid JOB_RUN_ID %q: must be a positive integer", args[0]))
		}

		run, err := w.Jobs.GetRun(ctx, jobs.GetRunRequest{RunId: runID})
		if err != nil {
			// The backend returns this when the run ID is unknown to the user.
			if errors.Is(err, apierr.ErrResourceDoesNotExist) {
				return renderError(ctx, cmd, "NOT_FOUND", "NOT_FOUND", false,
					fmt.Errorf("run %d not found: check the run ID and that it is a job run ID", runID))
			}
			return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
				fmt.Errorf("failed to get status for run %d: %w", runID, err))
		}

		workspaceID, err := w.CurrentWorkspaceID(ctx)
		if err != nil {
			return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
				fmt.Errorf("failed to get workspace id for run %d: %w", runID, err))
		}

		data := buildGetData(run)
		data.DashboardURL = dashboardURL(w.Config.Host, runID, workspaceID)
		ids := mlflowIDs(ctx, w, run)
		if ids != nil {
			url := mlflowLogsURL(w.Config.Host, ids)
			data.MLflowURL = &url
		}
		if task := findForEachTask(run); task != nil {
			data.Sweep = buildSweepInfo(ctx, w, task)
		}

		if root.OutputType(cmd) == flags.OutputText {
			out := cmd.OutOrStdout()
			addTextLinks(ctx, out, w, &data, ids)

			// Lead with the job run link (hyperlinked, falling back to the bare
			// URL off a terminal), then a gap before the training config and the
			// status table, mirroring the Python CLI's header.
			fmt.Fprintf(out, "Job Link: %s\n\n", hyperlink(ctx, out, data.DashboardURL, data.DashboardURL))

			// Text mode shows the training-config YAML before the status,
			// mirroring `air get run`. JSON output omits it.
			if path := yamlConfigPath(run); path != "" {
				printConfigYAML(ctx, out, w, path)
			}
		}
		return renderEnvelope(ctx, data)
	}

	return cmd
}

// buildGetData extracts the fields we display from a run. The text-table cells
// are pre-rendered here with their "N/A" fallbacks; the Run ID, Experiment, and
// MLflow Run cells are finalized later by addTextLinks once the dashboard and
// MLflow identifiers are known.
func buildGetData(run *jobs.Run) getData {
	data := getData{
		RunID:           strconv.FormatInt(run.RunId, 10),
		Status:          runStatus(run.State),
		StartedAt:       startedAt(run),
		DurationSeconds: durationSeconds(run),
		AttemptNumber:   latestAttemptNumber(run),
		ExperimentName:  experimentName(run),
	}
	data.RunIDDisplay = data.RunID
	data.SubmittedDisplay = submittedDisplay(run)
	data.DurationDisplay = na
	if data.DurationSeconds != nil {
		data.DurationDisplay = formatDuration(*data.DurationSeconds)
	}
	data.ExperimentDisplay = na
	if data.ExperimentName != nil {
		data.ExperimentDisplay = *data.ExperimentName
	}
	data.MLflowDisplay = na
	data.UserDisplay = orNA(run.CreatorUserName)
	data.AcceleratorsDisplay = orNA(accelerators(run))
	return data
}

// addTextLinks adds the terminal hyperlinks shown in text mode: the Run ID links
// to the run's dashboard page (Python embeds this on the Run ID instead of a
// separate Dashboard row), and the Experiment and MLflow Run cells link to their
// MLflow pages. On a non-terminal these degrade to plain text.
func addTextLinks(ctx context.Context, out io.Writer, w *databricks.WorkspaceClient, data *getData, ids *mlflowIdentifiers) {
	data.RunIDDisplay = hyperlink(ctx, out, data.RunID, data.DashboardURL)
	if ids == nil {
		return
	}
	if data.ExperimentName != nil {
		data.ExperimentDisplay = hyperlink(ctx, out, *data.ExperimentName, mlflowExperimentURL(w.Config.Host, ids))
	}
	data.MLflowDisplay = hyperlink(ctx, out, mlflowRunLabel(ctx, w, ids.RunID), mlflowRunURL(w.Config.Host, ids))
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

// printConfigYAML downloads the run's training-config YAML and writes it to out
// (stdout), mirroring the Python `air get`. It is best-effort: a download or read
// failure is surfaced as a warning on stderr but does not fail the command.
func printConfigYAML(ctx context.Context, out io.Writer, w *databricks.WorkspaceClient, path string) {
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

	fmt.Fprintln(out, "Training Configuration:")
	fmt.Fprintln(out, reformatYAMLForDisplay(content))
	fmt.Fprintln(out)
}
