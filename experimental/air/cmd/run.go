package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/shellquote"
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

	// cobra passes -h's positional args to the help func before Args/required-flag
	// validation, so a config path documents a field without needing -f.
	cmd.SetHelpFunc(func(c *cobra.Command, args []string) {
		fields := c.Flags().Args()
		if len(fields) == 0 {
			// Parent() is nil for a detached command (unit tests).
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

		jsonOut := root.OutputType(cmd) == flags.OutputJSON

		// Announce the experiment before uploading; skipped in JSON mode to keep
		// stdout a clean envelope stream.
		if !jsonOut {
			cmdio.LogString(ctx, "Submitting experiment: "+cfg.ExperimentName)
		}

		w := cmdctx.WorkspaceClient(ctx)
		runID, dashboardURL, err := submitWorkload(ctx, w, cfg, file, idempotencyKey, !jsonOut)
		if err != nil {
			return err
		}

		runIDStr := strconv.FormatInt(runID, 10)

		if !watch {
			if !jsonOut {
				out := cmd.OutOrStdout()
				printSubmitResult(ctx, out, runIDStr, dashboardURL)
				// Append the MLflow links only if they resolve; a bare submit is not
				// blocked on them since the confirmation above is already printed.
				if ids := resolveMLflowIDsForRun(ctx, w, runID); ids != nil {
					printMLflowLinks(ctx, out, w.Config.Host, ids)
				}
				printPostSubmitGuidance(out, w.Config.Profile, runIDStr)
				return nil
			}
			// PENDING is the submit status, distinct from the --watch JSONL
			// SUBMITTED event type below.
			return renderEnvelope(ctx, runResult{Status: "PENDING", RunID: runIDStr, DashboardURL: dashboardURL})
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

		watchCtx, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()

		if !jsonOut {
			out := cmd.OutOrStdout()
			// The MLflow links stream in via the logs below, so don't poll here.
			printSubmitResult(ctx, out, runIDStr, dashboardURL)
			// Separate the submit summary from the streamed logs.
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Monitoring run and streaming logs...")
			printLogsDivider(ctx, out)
			return handleWatchResult(out, w.Config.Profile, runIDStr, runLogs(watchCtx, cmd, req))
		}

		// --json: emit SUBMITTED first (so a consumer sees the run id immediately),
		// STATUS events on each lifecycle transition, and a closing terminal-status
		// envelope after streaming.
		out := cmd.OutOrStdout()
		printSubmittedEvent(out, runIDStr, dashboardURL)
		req.onStatusChange = func(current, previous string) {
			printStatusEvent(out, current, previous)
		}
		err = runLogs(watchCtx, cmd, req)

		// Re-resolve the run for the closing envelope. STATUS events only fire on
		// the Bricklens path, so the terminal status must come from the run's
		// actual state — correct whether Bricklens or the MLflow fallback served
		// the logs.
		printTerminalEvent(out, runIDStr, watchTerminalStatus(ctx, w, runID), dashboardURL)
		return err
	}

	return cmd
}

func airLogsCommand(profile, runID string) string {
	args := []string{"databricks", "experimental", "air", "logs"}
	if profile != "" {
		args = append(args, "-p", shellquote.BashArg(profile))
	}
	args = append(args, shellquote.BashArg(runID))
	return strings.Join(args, " ")
}

func printPostSubmitGuidance(out io.Writer, profile, runID string) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Stream logs in real time using:")
	fmt.Fprintln(out, "  "+airLogsCommand(profile, runID))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Tip: use --watch when submitting a run to monitor the job and stream logs in real time.")
}

func handleWatchResult(out io.Writer, profile, runID string, err error) error {
	if !errors.Is(err, context.Canceled) {
		return err
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Stopped streaming logs. The workload was not canceled.")
	fmt.Fprintln(out, "Resume streaming logs using:")
	fmt.Fprintln(out, "  "+airLogsCommand(profile, runID))
	return root.ErrAlreadyPrinted
}

// printSubmitResult writes the green success line and Job Run link. These don't
// depend on the MLflow IDs, so they print before any MLflow poll. The link is
// styled (blue, underlined) and clickable, matching the `air get` view, and
// degrades to plain text on non-rich terminals.
func printSubmitResult(ctx context.Context, out io.Writer, runIDStr, dashboardURL string) {
	renderer, colorOn := cmdio.NewRenderer(ctx, out)
	p := newPalette(renderer)

	fmt.Fprintln(out, p.green.Render("Submitted workload with Job Run ID: "+runIDStr))
	fmt.Fprintln(out, "View job run at: "+link(colorOn, p.blue, dashboardURL, dashboardURL))
}

// printMLflowLinks appends the styled, clickable MLflow run and experiment links
// once their IDs are resolved.
func printMLflowLinks(ctx context.Context, out io.Writer, host string, ids *mlflowIdentifiers) {
	renderer, colorOn := cmdio.NewRenderer(ctx, out)
	p := newPalette(renderer)

	runURL := mlflowRunURL(host, ids)
	expURL := mlflowExperimentURL(host, ids)
	fmt.Fprintln(out, "View MLflow run at: "+link(colorOn, p.blue, runURL, runURL))
	fmt.Fprintln(out, "View MLflow experiment at: "+link(colorOn, p.blue, expURL, expURL))
}

// logsDividerWidth is the total display width of the --watch logs divider.
const logsDividerWidth = 60

// printLogsDivider prints a centered "Logs" rule marking where the streamed
// --watch logs begin, separating them from the submit summary. The dim color is
// dropped on non-rich terminals; the rule characters are always printed.
func printLogsDivider(ctx context.Context, out io.Writer) {
	renderer, _ := cmdio.NewRenderer(ctx, out)
	p := newPalette(renderer)

	const label = " Logs "
	side := max((logsDividerWidth-utf8.RuneCountInString(label))/2, 0)
	rule := strings.Repeat("─", side) + label + strings.Repeat("─", side)
	fmt.Fprintln(out, p.n7.Render(rule))
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
