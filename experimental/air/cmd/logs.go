package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strconv"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

func newLogsCommand() *cobra.Command {
	var (
		node       int
		lines      int
		minutes    int
		retry      int
		downloadTo string
		review     bool
	)

	cmd := &cobra.Command{
		Use:   "logs JOB_RUN_ID",
		Args:  root.ExactArgs(1),
		Short: "Stream or fetch logs for a run",
		Long:  `Stream logs from an active run, or fetch logs from a completed run.`,
	}

	cmd.Flags().IntVar(&node, "node", 0, "Fetch logs from this node")
	cmd.Flags().IntVar(&lines, "lines", 0, "For completed runs, print the last N lines (default 10000)")
	cmd.Flags().IntVar(&minutes, "minutes", 0, "Fetch only logs from the last N minutes")
	cmd.Flags().IntVar(&retry, "retry", -1, "View logs from a specific retry attempt; -1 means latest")
	cmd.Flags().StringVar(&downloadTo, "download-to", "", "Download all logs to this directory instead of printing")
	cmd.Flags().BoolVar(&review, "review", false, "Download logs from all nodes and filter for error signatures")
	cmd.Flags().MarkHidden("review")

	// In -o json mode an auth failure should be a JSON error envelope, not a bare
	// error. ErrAlreadyPrinted passes through.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		err := root.MustWorkspaceClient(cmd, args)
		if err == nil || errors.Is(err, root.ErrAlreadyPrinted) {
			return err
		}
		return authError(cmd.Context(), cmd, err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// --review is not yet implemented; reject rather than silently ignore.
		if review {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				errors.New("--review is not implemented yet"))
		}

		// A download always writes the full log, so a tail or time window would be
		// silently dropped.
		if downloadTo != "" && (cmd.Flags().Changed("lines") || minutes > 0) {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				errors.New("--download-to writes complete logs, so it cannot be combined with --lines or --minutes"))
		}

		// --lines (line tail) and --minutes (time window) answer the same question
		// two ways, so reject both together rather than silently honoring one.
		if lines > 0 && minutes > 0 {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				errors.New("cannot combine --lines with --minutes: --lines tails by line count, --minutes by time window"))
		}
		if lines < 0 {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				fmt.Errorf("invalid --lines %d: must be positive", lines))
		}
		if minutes < 0 {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				fmt.Errorf("invalid --minutes %d: must be positive", minutes))
		}
		if node < 0 {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				fmt.Errorf("invalid --node %d: must not be negative", node))
		}
		if retry < -1 {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				fmt.Errorf("invalid --retry %d: must be -1 or greater", retry))
		}

		runID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil || runID <= 0 {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
				fmt.Errorf("invalid JOB_RUN_ID %q: must be a positive integer", args[0]))
		}

		// -1 signals "unset" (use the default cap); an explicit --lines 0 stays 0
		// and prints nothing.
		tailLines := -1
		if cmd.Flags().Changed("lines") {
			tailLines = lines
		}

		err = runLogs(ctx, cmd, logRequest{
			runID:         runID,
			node:          node,
			nodeSet:       cmd.Flags().Changed("node"),
			attempt:       retry,
			windowMinutes: minutes,
			tailLines:     tailLines,
			downloadTo:    downloadTo,
			jsonOutput:    root.OutputType(cmd) == flags.OutputJSON,
		})
		if downloadTo != "" || root.OutputType(cmd) == flags.OutputJSON {
			return err
		}
		return handleWatchResult(cmd.OutOrStdout(), cmdctx.WorkspaceClient(ctx).Config.Profile, args[0], err)
	}

	return cmd
}

// runLogs resolves the run, validates --retry, and fetches logs. It handles error
// reporting; the backend selection lives in fetchLogs.
func runLogs(ctx context.Context, cmd *cobra.Command, req logRequest) error {
	w := cmdctx.WorkspaceClient(ctx)

	// Validate credentials server-side before fetching (MustWorkspaceClient only
	// attaches them), so a bad token fails clearly here.
	if _, err := w.CurrentUser.Me(ctx, iam.MeRequest{}); err != nil {
		return authError(ctx, cmd, err)
	}

	status, err := resolveRunStatus(ctx, w, req.runID)
	if err != nil {
		if errors.Is(err, apierr.ErrResourceDoesNotExist) {
			return renderError(ctx, cmd, "NOT_FOUND", "NOT_FOUND", false,
				fmt.Errorf("run %d not found: check the run ID and that it is a job run ID", req.runID))
		}
		return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
			fmt.Errorf("failed to get status for run %d: %w", req.runID, err))
	}

	// -1 (default) means latest; reject an attempt past the newest.
	if req.attempt >= 0 && req.attempt > status.latestAttempt {
		return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false,
			fmt.Errorf("invalid retry %d: available retries are 0 to %d", req.attempt, status.latestAttempt))
	}

	// --download-to writes each node's logs to disk instead of streaming.
	if req.downloadTo != "" {
		success, err := downloadLogs(ctx, w, cmd.OutOrStdout(), req, status)
		if errors.Is(err, errNodeOutOfRange) {
			return renderError(ctx, cmd, "INVALID_ARGS", "PERMANENT", false, err)
		}
		if err != nil {
			return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
				fmt.Errorf("failed to download logs for run %d: %w", req.runID, err))
		}
		if !success {
			return root.ErrAlreadyPrinted
		}
		return nil
	}

	// A past retry of an active run has immutable logs: render once, don't follow.
	if req.attempt >= 0 && req.attempt < status.latestAttempt && !status.terminal() {
		req.staticView = true
	}

	out := cmd.OutOrStdout()
	success, err := fetchLogs(ctx, w, out, req, status)
	if err != nil {
		if errors.Is(err, apierr.ErrResourceDoesNotExist) {
			return renderError(ctx, cmd, "NOT_FOUND", "NOT_FOUND", false,
				fmt.Errorf("run %d not found: check the run ID and that it is a job run ID", req.runID))
		}
		return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
			fmt.Errorf("failed to fetch logs for run %d: %w", req.runID, err))
	}

	// A run that finished unsuccessfully exits non-zero; output was already
	// written, so don't reprint via Cobra.
	if !success {
		return root.ErrAlreadyPrinted
	}
	return nil
}

func resolveLogAttempt(run *jobs.Run, requested int) (int, int64, error) {
	if requested < -1 {
		return 0, 0, fmt.Errorf("invalid retry %d: must be -1 or greater", requested)
	}
	if len(run.Tasks) == 0 {
		return 0, 0, nil
	}
	latest := latestAttemptNumber(run)
	attempt := requested
	if attempt < 0 {
		attempt = latest
	}
	if attempt > latest {
		return 0, 0, fmt.Errorf("invalid retry %d: available retries are 0 to %d", requested, latest)
	}
	for _, v := range slices.Backward(run.Tasks) {
		if v.AttemptNumber == attempt {
			return attempt, v.RunId, nil
		}
	}
	return 0, 0, fmt.Errorf("run %d has no task for retry %d", run.RunId, attempt)
}

// fetchLogs serves logs from Bricklens, falling back to MLflow when Bricklens
// returns errBricklensFeatureDisabled.
func fetchLogs(ctx context.Context, w *databricks.WorkspaceClient, out io.Writer, req logRequest, status logRunStatus) (bool, error) {
	success, err := streamBricklensLogs(ctx, w, out, req, status)
	if errors.Is(err, errBricklensFeatureDisabled) {
		return mlflowLogFallback(ctx, w, out, req, status)
	}
	return success, err
}
