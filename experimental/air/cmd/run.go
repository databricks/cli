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

The workload is described by a YAML config file (see --file).

To look up a config field, pass its path to -h:

  databricks experimental air run -h config
  databricks experimental air run -h config.compute
  databricks experimental air run -h config.compute.accelerator_type

The path must be a separate argument: cobra reserves -h as a boolean, so
-h=config.compute and -hconfig.compute are not accepted.`,
	}

	// Document a config field instead of the command when -h is given a path.
	// cobra hands the help function the positional args it collected, and it does
	// so before Args and required-flag validation, so -f is not needed here.
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		fields := c.Flags().Args()
		if len(fields) == 0 {
			// Resolved through the parent rather than captured up front, so this
			// picks up the inherited help function instead of cobra's default.
			// A detached command (as in unit tests) has no parent to inherit from.
			if parent := c.Parent(); parent != nil {
				parent.HelpFunc()(c, args)
				return
			}
			_ = c.Usage()
			return
		}
		if err := writeConfigFieldHelp(c.OutOrStdout(), fields[0]); err != nil {
			c.PrintErrln("Error:", err)
		}
	})

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
