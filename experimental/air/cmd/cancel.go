package aircmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
)

// cancelData is the JSON payload printed by `air cancel`. `all` is set only for
// --all, `workspace` only when --all finds no active runs, and `failed` only
// when a run could not be cancelled.
type cancelData struct {
	Cancelled []string        `json:"cancelled"`
	All       bool            `json:"all,omitempty"`
	Workspace string          `json:"workspace,omitempty"`
	Failed    []cancelFailure `json:"failed,omitempty"`
}

type cancelFailure struct {
	RunID string `json:"run_id"`
	Error string `json:"error"`
}

func newCancelCommand() *cobra.Command {
	var (
		all bool
		yes bool
	)

	cmd := &cobra.Command{
		Use:   "cancel [JOB_RUN_ID...]",
		Short: "Cancel one or more runs",
		Long:  `Cancel one or more runs by ID, or cancel all of your active runs with --all.`,
	}

	cmd.Flags().BoolVar(&all, "all", false, "Cancel all of your active runs")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip the confirmation prompt")

	// Require exactly one of: one or more JOB_RUN_IDs, or --all. Cobra parses flags
	// before running this, so `all` reflects the user's input.
	cmd.Args = func(cmd *cobra.Command, args []string) error {
		switch {
		case all && len(args) > 0:
			return &root.InvalidArgsError{Command: cmd, Message: "cannot combine JOB_RUN_ID arguments with --all"}
		case !all && len(args) == 0:
			return &root.InvalidArgsError{Command: cmd, Message: "provide at least one JOB_RUN_ID, or use --all"}
		}
		return nil
	}

	// In -o json mode an auth failure should be a JSON error envelope, not a bare
	// error. ErrAlreadyPrinted passes through (already handled upstream).
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
		jsonOut := root.OutputType(cmd) == flags.OutputJSON

		runIDs := args
		data := cancelData{Cancelled: []string{}}

		if all {
			data.All = true

			me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
			if err != nil {
				return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
					fmt.Errorf("failed to resolve current user: %w", err))
			}
			host := strings.TrimRight(w.Config.Host, "/")

			if !jsonOut {
				cmdio.LogString(ctx, fmt.Sprintf("Searching active runs for %s in %s...", me.UserName, host))
			}

			// Fetch every active run (up to the scan bound) so --all cancels all
			// of them, not just the first page.
			fetcher := newRunFetcher(ctx, w, listQuery{activeOnly: true, userFilter: me.UserName})
			rows, err := fetcher.next(maxListScan)
			if err != nil {
				return renderError(ctx, cmd, "INTERNAL_ERROR", "TRANSIENT", true,
					fmt.Errorf("failed to list active runs: %w", err))
			}

			runIDs = make([]string, 0, len(rows))
			for i := range rows {
				if rows[i].RunID != "" {
					runIDs = append(runIDs, rows[i].RunID)
				}
			}

			if len(runIDs) == 0 {
				if jsonOut {
					data.Workspace = host
					return renderEnvelope(ctx, data)
				}
				cmdio.LogString(ctx, "No active runs found.")
				return nil
			}

			if !yes {
				displayCancelPreview(ctx, rows, host)
				confirmed, err := cmdio.AskYesOrNo(ctx, fmt.Sprintf("\nCancel %d run(s) in %s?", len(runIDs), host))
				if err != nil {
					return err
				}
				if !confirmed {
					cmdio.LogString(ctx, "Cancellation aborted.")
					return root.ErrAlreadyPrinted
				}
			}
		}

		for _, rid := range runIDs {
			err := cancelRun(ctx, w, rid)
			if err != nil {
				data.Failed = append(data.Failed, cancelFailure{RunID: rid, Error: err.Error()})
				if !jsonOut {
					if runNotFound(err) {
						cmdio.LogString(ctx, fmt.Sprintf("Run %s not found. Please check the run ID and ensure you're using a Job Run ID.", rid))
					} else {
						cmdio.LogString(ctx, fmt.Sprintf("Failed to cancel run %s: %s", rid, err))
					}
				}
				continue
			}
			data.Cancelled = append(data.Cancelled, rid)
			if !jsonOut {
				cmdio.LogString(ctx, "Successfully requested cancellation for run "+rid)
			}
		}

		if jsonOut {
			if err := renderEnvelope(ctx, data); err != nil {
				return err
			}
			// Print the envelope, but still exit non-zero on any failure.
			if len(data.Failed) > 0 {
				return root.ErrAlreadyPrinted
			}
			return nil
		}

		if len(data.Failed) > 0 {
			cmdio.LogString(ctx, fmt.Sprintf("%d run(s) failed to cancel.", len(data.Failed)))
			return root.ErrAlreadyPrinted
		}
		if all || len(data.Cancelled) > 1 {
			cmdio.LogString(ctx, fmt.Sprintf("Successfully requested cancellation for %d run(s).", len(data.Cancelled)))
		}
		return nil
	}

	return cmd
}

// runNotFound reports whether err means the run does not exist. The cancel
// endpoint returns 400 INVALID_PARAMETER_VALUE ("Run <id> does not exist") for
// an unknown run, and the SDK only remaps that to ErrResourceDoesNotExist for
// the runs/get path, not cancel — so we also detect the raw code here.
func runNotFound(err error) bool {
	if errors.Is(err, apierr.ErrResourceDoesNotExist) {
		return true
	}
	if apiErr, ok := errors.AsType[*apierr.APIError](err); ok {
		return apiErr.StatusCode == http.StatusBadRequest && apiErr.ErrorCode == "INVALID_PARAMETER_VALUE"
	}
	return false
}

// cancelRun requests cancellation of a single job run. The cancel is async, so
// the returned waiter is ignored.
func cancelRun(ctx context.Context, w *databricks.WorkspaceClient, rid string) error {
	runID, err := strconv.ParseInt(rid, 10, 64)
	if err != nil || runID <= 0 {
		return fmt.Errorf("invalid run ID %q: must be a positive integer", rid)
	}
	_, err = w.Jobs.CancelRun(ctx, jobs.CancelRun{RunId: runID})
	return err
}

// displayCancelPreview shows the runs that `cancel --all` is about to terminate.
func displayCancelPreview(ctx context.Context, rows []listRow, host string) {
	var sb strings.Builder
	fmt.Fprintf(&sb, "\nWorkspace: %s\n", host)
	fmt.Fprintf(&sb, "Found %d active run(s) to cancel:\n\n", len(rows))

	tw := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "Run ID\tExperiment\tStarted")
	for i := range rows {
		experiment := orNA(rows[i].Experiment)
		started := na
		if rows[i].StartedAt != nil {
			started = *rows[i].StartedAt
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", rows[i].RunID, experiment, started)
	}
	tw.Flush()

	cmdio.LogString(ctx, strings.TrimRight(sb.String(), "\n"))
}
