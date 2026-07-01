package aircmd

import (
	"context"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// maxListScan bounds how many runs `air list` inspects while looking for AIR runs
// that match the filters. runs/list returns runs of every kind, so this caps the
// work on a workspace with a large run history.
const maxListScan = 2000

// jobsPageLimit is the per-request page size for runs/list; enrichConcurrency
// bounds the parallel MLflow lookups.
const (
	jobsPageLimit     = 25
	enrichConcurrency = 8
)

// listData is the payload printed by `air list`.
type listData struct {
	Rows []listRow `json:"runs"`
}

// listRow is one run in the list. The json-tagged fields form the
// machine-readable output; fields tagged `json:"-"` are shown only in the
// human-readable table.
type listRow struct {
	RunID     string  `json:"run_id"`
	RunName   string  `json:"run_name"`
	User      string  `json:"user"`
	Status    string  `json:"status"`
	StartedAt *string `json:"started_at"`
	IsSweep   bool    `json:"is_sweep"`

	// Experiment, Duration, MLflowURL and Accelerators are table-only columns,
	// omitted from JSON to match `air list --json`.
	Experiment   string `json:"-"`
	Duration     string `json:"-"`
	MLflowURL    string `json:"-"`
	Accelerators string `json:"-"`
}

// listedRun pairs a row with its task run id, so the MLflow link can be fetched
// after the run has been filtered in.
type listedRun struct {
	row       listRow
	taskRunID int64
}

// listQuery holds the resolved inputs to a runFetcher.
type listQuery struct {
	activeOnly  bool
	userFilter  string
	filters     listFilters
	fetchMLflow bool
}

func newListCommand() *cobra.Command {
	var (
		limit    int
		active   bool
		allUsers bool
		filters  []string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Args:  root.NoArgs,
		Short: "List your recent runs (active and completed) for the current profile",
	}

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of runs to show")
	cmd.Flags().BoolVar(&active, "active", false, "Show only active runs")
	cmd.Flags().BoolVar(&allUsers, "all-users", false, "Show runs from all users")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter runs, e.g. experiment=foo* (repeatable)")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		w := cmdctx.WorkspaceClient(ctx)

		if limit <= 0 {
			return fmt.Errorf("invalid --limit %d: must be a positive integer", limit)
		}

		f, err := parseListFilters(filters)
		if err != nil {
			return err
		}

		// An explicit user= filter wins; otherwise default to the current user
		// unless --all-users is set. runs/list has no creator param, so the
		// creator is matched while scanning.
		userFilter := f.User
		if userFilter == "" && !allUsers {
			me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
			if err != nil {
				return fmt.Errorf("failed to resolve current user: %w", err)
			}
			userFilter = me.UserName
		}

		fetcher := newRunFetcher(ctx, w, listQuery{
			activeOnly:  active,
			userFilter:  userFilter,
			filters:     f,
			fetchMLflow: root.OutputType(cmd) == flags.OutputText,
		})

		// JSON prints the newest `limit` runs once. Text renders the table:
		// navigable in a terminal (paging in older runs on demand), printed once
		// when piped.
		if root.OutputType(cmd) == flags.OutputJSON {
			rows, err := fetcher.next(limit)
			if err != nil {
				return err
			}
			warnIfTruncated(ctx, fetcher)
			return renderEnvelope(ctx, listData{Rows: rows})
		}
		return renderListText(cmd, fetcher, limit)
	}

	return cmd
}

// runFetcher pages Jobs runs/list on demand, yielding AIR runs that match the
// user and filters. It buffers a page's leftover runs so successive next() calls
// resume where the last stopped — driving both one-shot output and lazy paging.
type runFetcher struct {
	ctx         context.Context
	w           *databricks.WorkspaceClient
	query       map[string]any
	userFilter  string
	filters     listFilters
	fetchMLflow bool

	pending   []jobRun // runs from the last page not yet inspected
	scanned   int
	exhausted bool
}

func newRunFetcher(ctx context.Context, w *databricks.WorkspaceClient, q listQuery) *runFetcher {
	query := map[string]any{
		"run_type":     "SUBMIT_RUN",
		"expand_tasks": true,
		"limit":        jobsPageLimit,
	}
	if q.activeOnly {
		query["active_only"] = true
	}
	return &runFetcher{
		ctx:         ctx,
		w:           w,
		query:       query,
		userFilter:  q.userFilter,
		filters:     q.filters,
		fetchMLflow: q.fetchMLflow,
	}
}

// next returns up to want more matching rows, paging runs/list (and buffering the
// leftover runs of a page) until it has enough, the server has no more pages, or
// it has scanned maxListScan runs. MLflow links are filled in for text output.
func (f *runFetcher) next(want int) ([]listRow, error) {
	var entries []listedRun
	for len(entries) < want {
		if len(f.pending) == 0 {
			if f.exhausted || f.scanned >= maxListScan {
				break
			}
			if err := f.fetchPage(); err != nil {
				return nil, err
			}
			continue
		}

		run := &f.pending[0]
		f.pending = f.pending[1:]
		f.scanned++

		if !isAirRun(run) {
			continue
		}
		if f.userFilter != "" && run.CreatorUserName != f.userFilter {
			continue
		}
		if !f.filters.matches(run) {
			continue
		}
		entries = append(entries, listedRun{row: buildListRow(run), taskRunID: taskRunID(run)})
	}
	if f.scanned >= maxListScan {
		f.exhausted = true
	}

	// MLflow links appear only in the text table, so the per-run get-output
	// lookups are skipped for JSON output (which omits the column anyway).
	if f.fetchMLflow {
		setMLflowLinks(f.ctx, f.w, entries)
	}

	rows := make([]listRow, len(entries))
	for i, e := range entries {
		rows[i] = e.row
	}
	return rows, nil
}

// fetchPage loads the next runs/list page into the pending buffer, marking the
// fetcher exhausted once the server reports no further pages.
func (f *runFetcher) fetchPage() error {
	resp, err := fetchJobRunsPage(f.ctx, f.w, f.query)
	if err != nil {
		return err
	}
	f.pending = resp.Runs
	if resp.NextPageToken == "" {
		f.exhausted = true
	} else {
		f.query["page_token"] = resp.NextPageToken
	}
	return nil
}

// warnIfTruncated logs when the scan hit maxListScan, so one-shot output signals
// its results may be incomplete.
func warnIfTruncated(ctx context.Context, f *runFetcher) {
	if f.scanned >= maxListScan {
		log.Warnf(ctx, "air list: stopped after scanning %d runs; results may be incomplete", maxListScan)
	}
}

// setMLflowLinks fills in each row's MLflow link in parallel, best-effort: a row
// whose IDs can't be resolved keeps its "-" placeholder.
func setMLflowLinks(ctx context.Context, w *databricks.WorkspaceClient, entries []listedRun) {
	var g errgroup.Group
	g.SetLimit(enrichConcurrency)
	for i := range entries {
		g.Go(func() error {
			if ids := mlflowIDsForTask(ctx, w, entries[i].taskRunID); ids != nil {
				entries[i].row.MLflowURL = mlflowLogsURL(w.Config.Host, ids)
			}
			return nil
		})
	}
	// mlflowIDsForTask never returns an error (it logs and yields nil), so Wait can't fail.
	_ = g.Wait()
}
