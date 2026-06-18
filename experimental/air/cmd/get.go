package aircmd

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
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

	// The fields below are pre-rendered text-view cells, excluded from JSON
	// (matching `air get run --json`). Each shows "N/A" when its value is
	// missing. The styled single-run renderer (render.go) consumes them; the
	// Run ID, Status, and MLflow Run cells it draws are styled and hyperlinked
	// there rather than stored here.
	SubmittedDisplay    string `json:"-"`
	DurationDisplay     string `json:"-"`
	ExperimentDisplay   string `json:"-"`
	UserDisplay         string `json:"-"`
	AcceleratorsDisplay string `json:"-"`
	EnvironmentDisplay  string `json:"-"`
	MaxRetriesDisplay   string `json:"-"`
	// Sweep replaces the single-run view for foreach runs.
	Sweep *sweepInfo `json:"-"`
}

// getTemplate is the text-mode layout for a sweep (foreach) run. Single runs are
// drawn by the styled renderer in render.go and never reach this template; it is
// used only when .Data.Sweep is set. It reads from the JSON envelope, so every
// field is reached through ".Data".
const getTemplate = `Sweep Run ID: {{.Data.RunID}}
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

		if root.OutputType(cmd) != flags.OutputText {
			return renderEnvelope(ctx, data)
		}

		out := cmd.OutOrStdout()
		if data.Sweep != nil {
			// A sweep has no single status, config, or timing, so lead with the
			// job run link and render the foreach summary table (getTemplate).
			fmt.Fprintf(out, "Job Link: %s\n\n", hyperlink(ctx, out, data.DashboardURL, data.DashboardURL))
			return renderEnvelope(ctx, data)
		}

		renderRunText(ctx, out, w, run, &data, ids)
		return nil
	}

	return cmd
}

// buildGetData extracts the fields we display from a run. The text-view cells
// are pre-rendered here with their "N/A" fallbacks; the styled renderer adds the
// hyperlinks and colors once the dashboard and MLflow identifiers are known.
func buildGetData(run *jobs.Run) getData {
	data := getData{
		RunID:           strconv.FormatInt(run.RunId, 10),
		Status:          runStatus(run.State),
		StartedAt:       startedAt(run),
		DurationSeconds: durationSeconds(run),
		AttemptNumber:   latestAttemptNumber(run),
		ExperimentName:  experimentName(run),
	}
	data.SubmittedDisplay = submittedDisplay(run)
	data.DurationDisplay = na
	if data.DurationSeconds != nil {
		data.DurationDisplay = formatDuration(*data.DurationSeconds)
	}
	data.ExperimentDisplay = na
	if data.ExperimentName != nil {
		data.ExperimentDisplay = *data.ExperimentName
	}
	data.UserDisplay = orNA(run.CreatorUserName)
	data.AcceleratorsDisplay = orNA(accelerators(run))
	data.EnvironmentDisplay = orNA(environment(run))
	data.MaxRetriesDisplay = maxRetries(run)
	return data
}
