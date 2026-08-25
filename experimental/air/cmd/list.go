package aircmd

import (
	"context"
	"fmt"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
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

	// Experiment, Duration, ETA, MLflowURL and Accelerators are table-only
	// columns, omitted from JSON to match `air list --json`.
	Experiment string `json:"-"`
	Duration   string `json:"-"`
	// ETA is a best-effort remaining-time estimate ("~48m 20s"), set only for a
	// running run we could estimate; empty renders as "-".
	ETA          string `json:"-"`
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
	allUsers    bool
	userFilter  string
	currentUser string
	filters     listFilters
	fetchMLflow bool
	limit       int
}

func newListCommand() *cobra.Command {
	var (
		limit     int
		allStatus bool
		allUsers  bool
		filters   []string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Args:  root.NoArgs,
		Short: "List your active runs for the current profile (use --all-status for finished runs)",
	}

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of runs to show")
	cmd.Flags().BoolVar(&allStatus, "all-status", false, "Show runs in all states (default: active only)")
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
		var currentUser string
		if userFilter == "" && !allUsers {
			me, err := w.CurrentUser.Me(ctx, iam.MeRequest{})
			if err != nil {
				return fmt.Errorf("failed to resolve current user: %w", err)
			}
			currentUser = me.UserName
			userFilter = currentUser
		}

		fetcher := newRunFetcher(ctx, w, listQuery{
			activeOnly:  !allStatus,
			allUsers:    allUsers,
			userFilter:  userFilter,
			currentUser: currentUser,
			filters:     f,
			fetchMLflow: root.OutputType(cmd) == flags.OutputText,
			limit:       limit,
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

// listStrategy is a source of matching runs, pulled in batches. Two implement it:
// jobsScanStrategy pages runs/list; indexStrategy hydrates the AiTrainingService
// index. The fetcher wraps whichever is chosen.
type listStrategy interface {
	// next returns up to want more matching runs (already row-built + task id).
	next(want int) ([]listedRun, error)
	// done reports whether the source has no more runs to yield.
	done() bool
	// truncated reports whether a safety cap stopped the scan short of the end.
	truncated() bool
}

// runFetcher yields matching rows in batches, driving both one-shot output and
// the interactive table's lazy paging. It wraps a listStrategy and adds the
// shared tail: MLflow enrichment (text only) and row projection.
type runFetcher struct {
	ctx         context.Context
	w           *databricks.WorkspaceClient
	fetchMLflow bool
	strategy    listStrategy

	exhausted bool
}

func newRunFetcher(ctx context.Context, w *databricks.WorkspaceClient, q listQuery) *runFetcher {
	return &runFetcher{
		ctx:         ctx,
		w:           w,
		fetchMLflow: q.fetchMLflow,
		strategy:    newListStrategy(ctx, w, q),
	}
}

// newListStrategy picks the fetch source. The AiTrainingService index serves only
// the caller's own runs, so it's used for an all-status self-scoped list; if the
// index load fails (e.g. endpoint unavailable in this workspace), we fall back to
// the Jobs scan so the command still returns. Everything else — the default
// active list, --all-users, and --all-status for another user — uses the scan.
func newListStrategy(ctx context.Context, w *databricks.WorkspaceClient, q listQuery) listStrategy {
	useIndex := !q.activeOnly && !q.allUsers && (q.userFilter == "" || q.userFilter == q.currentUser)
	if !useIndex {
		return newJobsScanStrategy(ctx, w, q)
	}
	idx := newIndexStrategy(ctx, w, q, q.limit)
	if err := idx.load(); err != nil {
		log.Debugf(ctx, "air list: AiTrainingService index unavailable, falling back to Jobs scan: %v", err)
		return newJobsScanStrategy(ctx, w, q)
	}
	return idx
}

// next pulls the next batch from the strategy, enriches it with MLflow links for
// text output, and projects it to rows. It sets exhausted once the strategy is
// drained so the interactive table knows to stop paging.
func (f *runFetcher) next(want int) ([]listRow, error) {
	entries, err := f.strategy.next(want)
	if err != nil {
		return nil, err
	}
	f.exhausted = f.strategy.done()

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

// jobsScanStrategy pages Jobs runs/list, keeping the AIR runs that match the user
// and filters. It buffers a page's leftover runs so successive next() calls
// resume where the last stopped.
type jobsScanStrategy struct {
	ctx        context.Context
	w          *databricks.WorkspaceClient
	iter       listing.Iterator[jobs.BaseRun]
	userFilter string
	filters    listFilters

	scanned int
}

func newJobsScanStrategy(ctx context.Context, w *databricks.WorkspaceClient, q listQuery) *jobsScanStrategy {
	req := jobs.ListRunsRequest{
		RunType:     jobs.RunTypeSubmitRun,
		ExpandTasks: true,
		Limit:       jobsPageLimit,
		ActiveOnly:  q.activeOnly,
	}
	return &jobsScanStrategy{
		ctx:        ctx,
		w:          w,
		iter:       w.Jobs.ListRuns(ctx, req),
		userFilter: q.userFilter,
		filters:    q.filters,
	}
}

func (s *jobsScanStrategy) next(want int) ([]listedRun, error) {
	var entries []listedRun
	for len(entries) < want && s.scanned < maxListScan && s.iter.HasNext(s.ctx) {
		base, err := s.iter.Next(s.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list runs: %w", err)
		}
		s.scanned++

		run := baseRunToRun(base)
		if !isAirRun(run) {
			continue
		}
		if s.userFilter != "" && run.CreatorUserName != s.userFilter {
			continue
		}
		if !s.filters.matches(run) {
			continue
		}
		entries = append(entries, listedRun{row: buildListRow(run), taskRunID: taskRunID(run)})
	}
	return entries, nil
}

func (s *jobsScanStrategy) done() bool {
	return s.scanned >= maxListScan || !s.iter.HasNext(s.ctx)
}

func (s *jobsScanStrategy) truncated() bool {
	return s.scanned >= maxListScan
}

// warnIfTruncated logs when a scan hit its safety cap, so one-shot output signals
// its results may be incomplete.
func warnIfTruncated(ctx context.Context, f *runFetcher) {
	if f.strategy.truncated() {
		log.Warnf(ctx, "air list: stopped after scanning %d runs; results may be incomplete", maxListScan)
	}
}

// setMLflowLinks fills in each row's MLflow link — and, for a running run, its
// ETA — in parallel, best-effort: a row whose IDs can't be resolved keeps its "-"
// placeholder, and a run we can't estimate simply has no ETA.
func setMLflowLinks(ctx context.Context, w *databricks.WorkspaceClient, entries []listedRun) {
	var g errgroup.Group
	g.SetLimit(enrichConcurrency)
	for i := range entries {
		g.Go(func() error {
			ids := mlflowIDsForTask(ctx, w, entries[i].taskRunID)
			if ids == nil {
				return nil
			}
			entries[i].row.MLflowURL = mlflowLogsURL(w.Config.Host, ids)
			// The ETA needs a progress-metric history fetch, so it's computed only
			// for running rows (the only ones that can have one).
			if entries[i].row.Status == string(jobs.RunLifeCycleStateRunning) {
				if eta := estimateTrainingETA(ctx, w, ids.RunID); eta != nil {
					entries[i].row.ETA = eta.compact()
				}
			}
			return nil
		})
	}
	// mlflowIDsForTask never returns an error (it logs and yields nil), so Wait can't fail.
	_ = g.Wait()
}
