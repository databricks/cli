package aircmd

import (
	"fmt"
	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
)

// runResult is the JSON payload for `air run`.
type runResult struct {
	Status       string `json:"status"`
	DryRun       bool   `json:"dry_run,omitempty"`
	RunID        string `json:"run_id,omitempty"`
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
		terminalStatus := "FAILED"
		req.onStatusChange = func(current, previous string) {
			printStatusEvent(out, current, previous)
			terminalStatus = current
		}
		err = runLogs(ctx, cmd, req)
		printTerminalEvent(out, runIDStr, terminalStatus, dashboardURL)
		return err
	}

	return cmd
}
