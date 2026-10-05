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
	"time"
	"unicode/utf8"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/databricks-sdk-go"
	"github.com/spf13/cobra"
)

// dryRunValidationTimeout bounds only the config:validate request.
const dryRunValidationTimeout = 15 * time.Second

// runResult is the JSON payload for `air run`.
type runResult struct {
	Status       string `json:"status"`
	DryRun       bool   `json:"dry_run,omitempty"`
	RunID        string `json:"run_id,omitempty"`
	DashboardURL string `json:"dashboard_url,omitempty"`
}

const experimentalContainersWarning = `+--------------------------------------------------------------------+
| WARNING: EXPERIMENTAL FEATURE                                      |
|                                                                    |
| This feature is experimental and may change without notice,        |
| including changes that could break existing workflows.             |
| Use it only if you accept these risks.                             |
+--------------------------------------------------------------------+`

func warnExperimentalContainers(ctx context.Context, cfg *runConfig) {
	if len(cfg.Containers) > 0 {
		cmdio.LogString(ctx, experimentalContainersWarning)
	}
}

// validateDryRunInWorkspace marks transient workspace access failures incomplete;
// configuration, authentication, and permission errors are returned directly.
func validateDryRunInWorkspace(ctx context.Context, cmd *cobra.Command, args []string, cfg *runConfig, idempotencyToken string, timeout time.Duration) (context.Context, error) {
	if !cmdctx.HasWorkspaceClient(ctx) {
		if err := root.MustWorkspaceClient(cmd, args); err != nil {
			return ctx, err
		}
		ctx = cmd.Context()
	}

	w := cmdctx.WorkspaceClient(ctx)
	_, funcDir, commandPath, err := prospectiveLaunchPaths(ctx, w, cfg)
	if err != nil {
		if !validationCouldNotComplete(err) {
			return ctx, err
		}
		return ctx, asIncompleteConfigValidation(err)
	}

	validationCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return ctx, validateConfig(validationCtx, w, cfg, commandPath, submittedContainers(cfg, funcDir), idempotencyToken)
}

func newRunCommand() *cobra.Command {
	return newRunCommandWithValidationTimeout(dryRunValidationTimeout)
}

func newRunCommandWithValidationTimeout(validationTimeout time.Duration) *cobra.Command {
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

  databricks air run -h config
  databricks air run -h config.compute
  databricks air run -h config.compute.accelerator_type

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
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Validate the config against the workspace without submitting a run")
	cmd.Flags().StringVar(&idempotencyKey, "idempotency-key", "", "Return the existing run if this key was already used")
	_ = cmd.MarkFlagRequired("file")

	// --dry-run authenticates after local parsing and validation so syntax, unknown
	// fields, and invalid overrides remain actionable without valid credentials.
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
			idempotencyToken := effectiveIdempotencyToken(idempotencyKey, cfg)
			ctx, validationErr := validateDryRunInWorkspace(ctx, cmd, args, cfg, idempotencyToken, validationTimeout)
			if validationErr != nil && !configValidationIncomplete(validationErr) {
				return validationErr
			}
			if err := validateIdempotencyToken(idempotencyToken); err != nil {
				return err
			}
			if validationErr != nil {
				cmdio.LogString(ctx, fmt.Sprintf("Warning: workspace validation could not be completed (%s); only local validation was performed.", validationErr))
			}
			if root.OutputType(cmd) == flags.OutputText {
				if validationErr != nil {
					cmdio.LogString(ctx, fmt.Sprintf("Dry run: local validation passed for %q; not submitting.", cfg.ExperimentName))
					return nil
				}
				cmdio.LogString(ctx, fmt.Sprintf("Dry run: configuration for %q is valid; not submitting.", cfg.ExperimentName))
				return nil
			}
			return renderEnvelope(ctx, runResult{Status: "DRY_RUN_OK", DryRun: true})
		}

		warnExperimentalContainers(ctx, cfg)

		jsonOut := root.OutputType(cmd) == flags.OutputJSON
		w := cmdctx.WorkspaceClient(ctx)

		// Announce the experiment before uploading; skipped in JSON mode to keep
		// stdout a clean envelope stream.
		if !jsonOut {
			cmdio.LogString(ctx, "Submitting experiment: "+cfg.ExperimentName)
		}

		runID, dashboardURL, err := submitWorkload(ctx, w, cfg, file, idempotencyKey, !jsonOut)
		if err != nil {
			return err
		}

		runIDStr := strconv.FormatInt(runID, 10)

		if !watch {
			if !jsonOut {
				out := cmd.OutOrStdout()
				printSubmitResult(ctx, out, runIDStr, dashboardURL)
				printPostSubmitGuidance(out, w.Config.Profile, runIDStr)
				grantSubmittedPermissions(ctx, w, runID, cfg.Permissions, true)
				return nil
			}
			// PENDING is the submit status, distinct from the --watch JSONL
			// SUBMITTED event type below.
			if err := renderEnvelope(ctx, runResult{Status: "PENDING", RunID: runIDStr, DashboardURL: dashboardURL}); err != nil {
				return err
			}
			grantSubmittedPermissions(ctx, w, runID, cfg.Permissions, false)
			return nil
		}

		// --watch: stream the submitted run's logs until it reaches a terminal
		// state, then exit with the run's outcome. This is the same pipeline as
		// `air logs <run>` (Bricklens with MLflow fallback).
		maxRetries := cfg.maxRetries()
		req := logRequest{
			runID:        runID,
			attempt:      -1,
			tailLines:    -1,
			jsonOutput:   jsonOut,
			maxRetries:   &maxRetries,
			retryTracker: newRetryTracker(logRunStatus{}),
		}

		watchCtx, stop := notifyInterrupt(ctx)
		defer stop()

		if !jsonOut {
			out := cmd.OutOrStdout()
			perNode, err := gpusPerNode(gpuType(cfg.Compute.AcceleratorType))
			if err != nil {
				return err
			}
			monitoringMessage := "Monitoring run and streaming logs..."
			if cfg.Compute.NumAccelerators > perNode {
				monitoringMessage = fmt.Sprintf("Monitoring run and streaming logs from node 0 of %d...", cfg.Compute.NumAccelerators/perNode)
			}
			// The MLflow links stream in via the logs below, so don't poll here.
			printSubmitResult(ctx, out, runIDStr, dashboardURL)
			grantSubmittedPermissions(ctx, w, runID, cfg.Permissions, true)
			// Separate the submit summary from the streamed logs.
			fmt.Fprintln(out)
			fmt.Fprintln(out, monitoringMessage)
			if maxRetries > 0 {
				unit := "times"
				if maxRetries == 1 {
					unit = "time"
				}
				fmt.Fprintf(out, "Failed attempts will be retried up to %d %s.\n", maxRetries, unit)
			}
			printLogsDivider(ctx, out)
			return handleWatchResult(out, w.Config.Profile, runIDStr, runLogs(watchCtx, cmd, req))
		}

		// --json: emit SUBMITTED first (so a consumer sees the run id immediately),
		// STATUS events on each lifecycle transition, and a closing terminal-status
		// envelope after streaming.
		out := cmd.OutOrStdout()
		printSubmittedEvent(out, runIDStr, dashboardURL)
		grantSubmittedPermissions(ctx, w, runID, cfg.Permissions, false)
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
	args := []string{"databricks", "air", "logs", shellquote.BashArg(runID)}
	if profile != "" {
		args = append(args, "-p", shellquote.BashArg(profile))
	}
	return strings.Join(args, " ")
}

func airGetCommand(profile, runID string) string {
	args := []string{"databricks", "air", "get", shellquote.BashArg(runID)}
	if profile != "" {
		args = append(args, "-p", shellquote.BashArg(profile))
	}
	return strings.Join(args, " ")
}

func printPostSubmitGuidance(out io.Writer, profile, runID string) {
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Tip: use --watch when submitting a run to stream logs to your terminal.")
	fmt.Fprintln(out, "Stream logs after submission using:")
	fmt.Fprintln(out, "  "+airLogsCommand(profile, runID))
}

// notifyInterrupt returns a context cancelled on the first interrupt (Ctrl-C),
// which lets the log stream unwind and print resume guidance via
// handleWatchResult; the CLI root installs no signal handler of its own. The
// caller must defer the returned stop.
//
// signal.Notify disables the default SIGINT disposition process-wide for as long
// as the channel stays registered, so a second Ctrl-C would merely be buffered
// and dropped, leaving no way to abort a hung teardown. Calling signal.Stop
// before cancel restores SIG_DFL first, so the cancellation is only observable
// once a second signal is guaranteed to terminate the process.
func notifyInterrupt(parent context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(parent)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		// Selecting on ctx.Done() too lets the goroutine exit on the normal (no
		// signal) path rather than parking on sigCh for the life of the process.
		select {
		case <-sigCh:
			signal.Stop(sigCh)
			cancel()
		case <-ctx.Done():
			signal.Stop(sigCh)
		}
	}()
	return ctx, cancel
}

func handleWatchResult(out io.Writer, profile, runID string, err error) error {
	if !errors.Is(err, context.Canceled) {
		return err
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Streaming logs interrupted.")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "To check status:")
	fmt.Fprintln(out, airGetCommand(profile, runID))
	fmt.Fprintln(out)
	fmt.Fprintln(out, "To resume streaming logs:")
	fmt.Fprintln(out, airLogsCommand(profile, runID))
	return root.ErrAlreadyPrinted
}

// printSubmitResult writes the green success line and Job Run link. The link is
// styled (blue, underlined) and clickable, matching the `air get` view, and
// degrades to plain text on non-rich terminals.
func printSubmitResult(ctx context.Context, out io.Writer, runIDStr, dashboardURL string) {
	renderer, colorOn := cmdio.NewRenderer(ctx, out)
	p := newPalette(renderer)

	fmt.Fprintln(out, p.green.Render("Submitted workload with Job Run ID: "+runIDStr))
	fmt.Fprintln(out, "View job run at: "+link(colorOn, p.blue, dashboardURL, dashboardURL))
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
