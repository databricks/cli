package ai

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

// statusData is the payload printed by `ai status`. The json-tagged fields form
// the machine-readable output; fields tagged `json:"-"` are shown only in the
// human-readable text view.
type statusData struct {
	RunID           string  `json:"run_id"`
	Status          string  `json:"status"`
	StartedAt       *string `json:"started_at"`
	DurationSeconds *int64  `json:"duration_seconds"`
	AttemptNumber   int     `json:"attempt_number"`
	ExperimentName  *string `json:"experiment_name"`
	DashboardURL    string  `json:"dashboard_url"`

	// Duration is the human-readable form of DurationSeconds, e.g. "12m 3s".
	Duration string `json:"-"`
	// Accelerators describes the run's GPUs, e.g. "8x H100".
	Accelerators string `json:"-"`
}

// statusTemplate is the text-mode layout. It reads from the JSON envelope, so
// every field is reached through ".Data". Optional rows are hidden when empty.
const statusTemplate = `Run ID:       {{.Data.RunID}}
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
{{- if .Data.Accelerators}}
Accelerators: {{.Data.Accelerators}}
{{- end}}
Dashboard:    {{.Data.DashboardURL}}
`

func newStatusCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status RUN_ID",
		Args:  root.ExactArgs(1),
		Short: "Show status and configuration for a run",
		Annotations: map[string]string{
			"template": statusTemplate,
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

		return renderEnvelope(ctx, buildStatusData(run))
	}

	return cmd
}

// buildStatusData extracts the fields we display from a run.
func buildStatusData(run *jobs.Run) statusData {
	data := statusData{
		RunID:           strconv.FormatInt(run.RunId, 10),
		Status:          runStatus(run.State),
		StartedAt:       startedAt(run),
		DurationSeconds: durationSeconds(run),
		AttemptNumber:   latestAttemptNumber(run),
		ExperimentName:  experimentName(run),
		DashboardURL:    run.RunPageUrl,
		Accelerators:    accelerators(run),
	}
	if data.DurationSeconds != nil {
		data.Duration = formatDuration(*data.DurationSeconds)
	}
	return data
}
