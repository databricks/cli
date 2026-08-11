package aircmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
)

// runResult is the JSON payload for `air run`.
type runResult struct {
	Status       string `json:"status"`
	DryRun       bool   `json:"dry_run,omitempty"`
	RunID        string `json:"run_id,omitempty"`
	JobID        string `json:"job_id,omitempty"`
	DashboardURL string `json:"dashboard_url,omitempty"`
}

func newRunCommand() *cobra.Command {
	var (
		file           string
		watch          bool
		overrides      []string
		dryRun         bool
		idempotencyKey string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Args:  root.NoArgs,
		Short: "Submit a training workload from a YAML config",
		Long: `Submit a training workload to Databricks serverless GPU compute.

The workload is described by a YAML config file (see --file).`,
	}

	cmd.Flags().StringVarP(&file, "file", "f", "", "Path to the workload YAML config")
	cmd.Flags().BoolVar(&watch, "watch", false, "Stream logs until the run completes")
	cmd.Flags().StringArrayVar(&overrides, "override", nil, "Override a YAML field, e.g. compute.num_accelerators=8 (repeatable)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate the config without submitting")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Return the existing run if this key was already used")
	_ = cmd.MarkFlagRequired("file")

	// --dry-run only validates the config locally, so it needs no workspace.
	// Submission requires an authenticated client.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if dryRun {
			return nil
		}
		return root.MustWorkspaceClient(cmd, args)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		cfg, err := loadRunConfigWithOverrides(ctx, file, overrides)
		if err != nil {
			return err
		}

		if dryRun {
			if root.OutputType(cmd) == flags.OutputText {
				cmdio.LogString(ctx, fmt.Sprintf("Dry run: configuration for %q is valid; not submitting.", cfg.ExperimentName))
				return nil
			}
			return renderEnvelope(ctx, runResult{Status: "DRY_RUN_OK", DryRun: true})
		}

		// A schedule turns the workload into a persistent, scheduled job instead of a
		// one-time run: create (or update) the job and return, since there is no
		// immediate run to submit or stream.
		if cfg.Schedule != nil {
			return runScheduled(ctx, cmd, cfg, file)
		}

		w := cmdctx.WorkspaceClient(ctx)
		runID, dashboardURL, err := submitWorkload(ctx, w, cfg, file, idempotencyKey)
		if err != nil {
			return err
		}

		runIDStr := strconv.FormatInt(runID, 10)
		jsonOut := root.OutputType(cmd) == flags.OutputJSON

		if !watch {
			if !jsonOut {
				cmdio.LogString(ctx, "Submitted run "+runIDStr)
				cmdio.LogString(ctx, "View at: "+dashboardURL)
				cmdio.LogString(ctx, "\nTip: use --watch to stream logs until the run completes.")
				return nil
			}
			return renderEnvelope(ctx, runResult{Status: "SUBMITTED", RunID: runIDStr, DashboardURL: dashboardURL})
		}

		// --watch: stream the submitted run's logs until it reaches a terminal
		// state, then exit with the run's outcome. This is the same pipeline as
		// `air logs <run>` (Bricklens with MLflow fallback).
		req := logRequest{
			runID:      runID,
			attempt:    -1,
			tailLines:  -1,
			jsonOutput: jsonOut,
		}

		if !jsonOut {
			cmdio.LogString(ctx, "Submitted run "+runIDStr)
			cmdio.LogString(ctx, "View at: "+dashboardURL)
			cmdio.LogString(ctx, "Monitoring run and streaming logs...")
			return runLogs(ctx, cmd, req)
		}

		// --json: emit SUBMITTED first (so a consumer sees the run id immediately),
		// STATUS events on each lifecycle transition, and a closing terminal-status
		// envelope after streaming. Mirrors the Python CLI's --watch JSONL contract.
		out := cmd.OutOrStdout()
		printSubmittedEvent(out, runIDStr, dashboardURL)
		req.onStatusChange = func(current, previous string) {
			printStatusEvent(out, current, previous)
		}
		err = runLogs(ctx, cmd, req)

		// Re-resolve the run for the closing envelope. STATUS events only fire on
		// the Bricklens path, so the terminal status must come from the run's
		// actual state — correct whether Bricklens or the MLflow fallback served
		// the logs.
		printTerminalEvent(out, runIDStr, watchTerminalStatus(ctx, w, runID), dashboardURL)
		return err
	}

	return cmd
}

// runScheduled creates (or updates) a persistent, scheduled job for a workload
// whose config carries a `schedule`. Unlike a submit, there is no immediate run
// to stream, so --watch does not apply here.
func runScheduled(ctx context.Context, cmd *cobra.Command, cfg *runConfig, configPath string) error {
	w := cmdctx.WorkspaceClient(ctx)
	jobID, jobURL, created, err := createScheduledJob(ctx, w, cfg, configPath)
	if err != nil {
		return err
	}
	jobIDStr := strconv.FormatInt(jobID, 10)

	if root.OutputType(cmd) == flags.OutputJSON {
		status := "SCHEDULED_UPDATED"
		if created {
			status = "SCHEDULED_CREATED"
		}
		return renderEnvelope(ctx, runResult{Status: status, JobID: jobIDStr, DashboardURL: jobURL})
	}

	verb := "Updated"
	if created {
		verb = "Created"
	}
	cmdio.LogString(ctx, fmt.Sprintf("%s scheduled job %s", verb, jobIDStr))
	cmdio.LogString(ctx, "View at: "+jobURL)
	cmdio.LogString(ctx, fmt.Sprintf("Runs on schedule: %s (%s)", cfg.Schedule.QuartzCronExpression, cfg.Schedule.TimezoneID))
	if cfg.Schedule.PauseStatus == "PAUSED" {
		cmdio.LogString(ctx, "The schedule is PAUSED; set pause_status: UNPAUSED (or unpause it in the Jobs UI) to activate it.")
	}
	return nil
}

// watchTerminalStatus resolves a watched run's final display state for the
// closing --watch envelope. The run is terminal once streaming returns; if the
// status can't be re-fetched, "UNKNOWN" is reported rather than guessing.
func watchTerminalStatus(ctx context.Context, w *databricks.WorkspaceClient, runID int64) string {
	status, err := resolveRunStatus(ctx, w, runID)
	if err != nil {
		return "UNKNOWN"
	}
	return status.displayState()
}
