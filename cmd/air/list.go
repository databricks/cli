package aircmd

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

// maxListScan bounds how many runs `air list` inspects while looking for AIR runs
// that match the filters. runs/list returns runs of every kind, so this caps the
// work on a workspace with a large run history.
const maxListScan = 2000

// maxFallbackListScan bounds the Jobs scan that stands in for an unavailable
// AiTrainingService index. That scan can't use active_only, so it walks the full
// run history, which costs several seconds per page on a busy workspace.
const maxFallbackListScan = 250

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
	RunID           string  `json:"run_id"`
	RunName         string  `json:"run_name"`
	User            string  `json:"user"`
	Status          string  `json:"status"`
	DisplayStatus   string  `json:"-"`
	StartedAt       *string `json:"started_at"`
	IsSweep         bool    `json:"is_sweep"`
	ExperimentName  *string `json:"experiment_name"`
	DurationSeconds *int64  `json:"duration_seconds"`
	AcceleratorType *string `json:"accelerator_type"`
	NumAccelerators *int    `json:"num_accelerators"`
	DashboardURL    string  `json:"dashboard_url"`

	// The fields below are pre-rendered table cells or text-only enrichment.
	Experiment string `json:"-"`
	Duration   string `json:"-"`
	// Progress is a best-effort estimate ("45% · ~2h 15m"), set only for a running
	// run we could estimate; empty renders as "-".
	Progress      string `json:"-"`
	MLflowURL     string `json:"-"`
	MLflowLabel   string `json:"-"`
	RunURL        string `json:"-"`
	ExperimentURL string `json:"-"`
	Accelerators  string `json:"-"`
}

// listedRun pairs a row with its parent run id and task runs, so the MLflow link
// can be fetched after the run has been filtered in and cached afterward.
type listedRun struct {
	row      listRow
	runID    int64
	tasks    []jobs.RunTask // see retryTasks
	terminal bool
}

// listQuery holds the resolved inputs to a runFetcher.
type listQuery struct {
	activeOnly bool
	allUsers   bool
	userFilter string
	// currentUser resolves the caller's user name for a self-scoped list. It is
	// nil when an explicit user= filter or --all-users is set.
	currentUser func() (string, error)
	// selfUser is the caller's user name when it was resolved up front, despite
	// an explicit user= filter (--all-status only). The strategy compares it
	// with userFilter to tell a self filter from a filter for another user.
	selfUser    string
	filters     listFilters
	fetchMLflow bool
	limit       int
	workspaceID int64
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
		Short: "List your active runs",
		Long: `List active runs for the current user.

Filters:
  Use --filter KEY=VALUE per condition. A run must match every condition.

  accelerator_type  Accelerator family or full YAML type. Examples:
                    A10, H100, GPU_1xA10, GPU_8xH100.
                    Workspace types: databricks air run -h config.compute.accelerator_type
  experiment        Case-insensitive glob matched against the experiment name.
  num_accelerators  Total GPU count.
  user              Run creator identifier.`,
		Example: `  databricks air list --filter accelerator_type=H100

  databricks air list --filter accelerator_type=GPU_8xH100

  databricks air list --all-status \
    --filter 'experiment=qwen*' \
    --filter num_accelerators=8 \
    --filter user=person@example.com`,
	}

	cmd.PreRunE = root.MustWorkspaceClient

	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of runs to show")
	cmd.Flags().BoolVar(&allStatus, "all-status", false, "Show runs in all states (default: active only)")
	cmd.Flags().BoolVar(&allUsers, "all-users", false, "Show runs from all users")
	cmd.Flags().StringArrayVar(&filters, "filter", nil, "Filter runs by KEY=VALUE (repeatable)")

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
		if f.AcceleratorType != "" {
			configureAcceleratorFilter(&f)
		}

		// An explicit user= filter wins; otherwise default to the current user
		// unless --all-users is set. The user is resolved only if a strategy
		// needs it: the index is already scoped to the caller, and runs/list has
		// no creator param, so the scan matches the creator itself. With
		// --all-status the current user is also resolved for an explicit user=
		// filter, so filtering on your own email uses the index instead of a
		// full-history scan.
		var currentUser func() (string, error)
		var selfUser string
		if !allUsers && (f.User == "" || allStatus) {
			if f.User == "" {
				currentUser = sync.OnceValues(func() (string, error) { return currentUserEmail(ctx, w) })
			} else {
				// Resolve the caller up front so the strategy can tell a self
				// filter from a filter for another user. A lookup failure keeps
				// the scan, which matches the creator itself.
				user, err := currentUserEmail(ctx, w)
				if err != nil {
					log.Debugf(ctx, "air list: could not resolve current user, scanning Jobs runs for user=%s: %v", f.User, err)
				} else {
					selfUser = user
				}
			}
		}

		fetcher := newRunFetcher(ctx, w, listQuery{
			activeOnly:  !allStatus,
			allUsers:    allUsers,
			userFilter:  f.User,
			currentUser: currentUser,
			selfUser:    selfUser,
			filters:     f,
			fetchMLflow: root.OutputType(cmd) == flags.OutputText,
			limit:       limit,
			workspaceID: workspaceIDForRunLink(ctx, w),
		})
		if f.AcceleratorType != "" {
			fetcher.validateEmpty = func() error {
				return validateEmptyAcceleratorFilter(ctx, w, f.AcceleratorType)
			}
		}

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
		return renderListText(cmd, fetcher, limit, !allStatus)
	}

	return cmd
}

func validateEmptyAcceleratorFilter(ctx context.Context, w *databricks.WorkspaceClient, value string) error {
	options, err := listWorkspaceComputeOptions(ctx, w)
	if err != nil {
		log.Debugf(ctx, "air list: could not validate accelerator filter against workspace compute options: %v", err)
		return fmt.Errorf("no runs matched --filter accelerator_type=%q\nRetry or run 'databricks air run -h config.compute.accelerator_type' to check available types", value)
	}
	return validateAcceleratorFilter(value, options)
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
	ctx           context.Context
	w             *databricks.WorkspaceClient
	fetchMLflow   bool
	strategy      listStrategy
	workspaceID   int64
	validateEmpty func() error

	exhausted      bool
	matchedAny     bool
	validatedEmpty bool
}

func newRunFetcher(ctx context.Context, w *databricks.WorkspaceClient, q listQuery) *runFetcher {
	return &runFetcher{
		ctx:         ctx,
		w:           w,
		fetchMLflow: q.fetchMLflow,
		strategy:    newListStrategy(ctx, w, q),
		workspaceID: q.workspaceID,
	}
}

// newListStrategy picks the fetch source. The AiTrainingService index serves only
// the caller's own runs, so it's used for an all-status self-scoped list; if the
// index load fails (e.g. endpoint unavailable in this workspace), we fall back to
// a bounded Jobs scan so the command still returns. Everything else — the default
// active list, --all-users, and --all-status for another user — uses the scan.
func newListStrategy(ctx context.Context, w *databricks.WorkspaceClient, q listQuery) listStrategy {
	// The index serves only the caller's own runs: a list with no user filter
	// (the scan would resolve the caller itself), or an explicit user= filter
	// that matches the caller case-insensitively (resolved up front under
	// --all-status).
	isSelf := q.userFilter == "" || (q.selfUser != "" && strings.EqualFold(q.userFilter, q.selfUser))
	useIndex := !q.activeOnly && !q.allUsers && isSelf
	if !useIndex {
		return newJobsScanStrategy(ctx, w, q, maxListScan)
	}
	idx := newIndexStrategy(ctx, w, q, q.limit)
	if err := idx.load(); err != nil {
		log.Warnf(ctx, "air list: the AI Runtime run index is unavailable (%v); falling back to a slower scan of up to %d recent job runs", err, maxFallbackListScan)
		return newJobsScanStrategy(ctx, w, q, maxFallbackListScan)
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
	if len(entries) > 0 {
		f.matchedAny = true
	}
	if len(entries) == 0 && f.exhausted && !f.matchedAny && f.validateEmpty != nil && !f.validatedEmpty {
		f.validatedEmpty = true
		if err := f.validateEmpty(); err != nil {
			return nil, err
		}
	}

	// MLflow links appear only in the text table, so the per-run get-output
	// lookups are skipped for JSON output (which omits the column anyway).
	if f.fetchMLflow {
		setMLflowLinks(f.ctx, f.w, f.w.Config.Host, entries)
		if s, ok := f.strategy.(*indexStrategy); ok {
			s.cacheMLflowLinks(entries)
		}
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
	ctx         context.Context
	w           *databricks.WorkspaceClient
	iter        listing.Iterator[jobs.BaseRun]
	userFilter  string
	currentUser func() (string, error) // resolves userFilter for a self-scoped scan
	filters     listFilters
	host        string
	workspaceID int64

	// maxScan caps how many runs this scan inspects; see maxListScan.
	maxScan int
	scanned int
}

func newJobsScanStrategy(ctx context.Context, w *databricks.WorkspaceClient, q listQuery, maxScan int) *jobsScanStrategy {
	req := jobs.ListRunsRequest{
		RunType:     jobs.RunTypeSubmitRun,
		ExpandTasks: true,
		Limit:       jobsPageLimit,
		ActiveOnly:  q.activeOnly,
	}
	// Resolve the current user while the first runs/list page is fetched; next()
	// waits for it only when it first needs to match a run's creator.
	if q.currentUser != nil {
		go func() { _, _ = q.currentUser() }()
	}
	return &jobsScanStrategy{
		ctx:         ctx,
		w:           w,
		iter:        w.Jobs.ListRuns(ctx, req),
		userFilter:  q.userFilter,
		currentUser: q.currentUser,
		filters:     q.filters,
		host:        w.Config.Host,
		workspaceID: q.workspaceID,
		maxScan:     maxScan,
	}
}

func (s *jobsScanStrategy) next(want int) ([]listedRun, error) {
	var entries []listedRun
	for len(entries) < want && s.scanned < s.maxScan && s.iter.HasNext(s.ctx) {
		base, err := s.iter.Next(s.ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list runs: %w", err)
		}
		s.scanned++

		run := baseRunToRun(base)
		if !isAirRun(run) {
			continue
		}
		if s.currentUser != nil {
			user, err := s.currentUser()
			if err != nil {
				return nil, err
			}
			s.userFilter, s.currentUser = user, nil
		}
		if s.userFilter != "" && run.CreatorUserName != s.userFilter {
			continue
		}
		if !s.filters.matches(run) {
			continue
		}
		entries = append(entries, listedRun{row: buildListRow(run, s.host, s.workspaceID), runID: run.RunId, tasks: retryTasks(run), terminal: isTerminal(run)})
	}
	return entries, nil
}

func (s *jobsScanStrategy) done() bool {
	return s.scanned >= s.maxScan || !s.iter.HasNext(s.ctx)
}

func (s *jobsScanStrategy) truncated() bool {
	return s.scanned >= s.maxScan
}

// warnIfTruncated warns when a scan hit its safety cap, so the output signals
// its results may be incomplete.
func warnIfTruncated(ctx context.Context, f *runFetcher) {
	if !f.strategy.truncated() {
		return
	}
	// Only the scan strategy can report truncated, and it carries the scanned
	// count for the warning.
	s, ok := f.strategy.(*jobsScanStrategy)
	if !ok {
		return
	}
	log.Warnf(ctx, "air list: stopped after scanning %d runs; results may be incomplete. Narrow the search with --filter or a smaller --limit", s.scanned)
}

// setMLflowLinks fills in each row's MLflow link, label, and experiment URL -
// and, for a running run, its ETA - in parallel, best-effort: a row whose IDs
// can't be resolved keeps its "-" placeholder, and a run we can't estimate has
// no ETA.
func setMLflowLinks(ctx context.Context, w *databricks.WorkspaceClient, host string, entries []listedRun) {
	var g errgroup.Group
	g.SetLimit(enrichConcurrency)
	for i := range entries {
		g.Go(func() error {
			if entries[i].row.MLflowURL != "" && entries[i].row.MLflowURL != "-" {
				return nil
			}
			ids := mlflowIDsForRetries(ctx, w, entries[i].tasks, entries[i].terminal)
			if ids == nil {
				return nil
			}
			entries[i].row.MLflowURL = mlflowLogsURL(host, ids)
			name := fetchMLflowRunName(ctx, w, ids.RunID)
			// Shorten the label, not the retry note, to fit the column.
			note := ids.retryNote()
			entries[i].row.MLflowLabel = mlflowRunLabel(name, ids.RunID, mlflowColWidth-len(note)) + note
			entries[i].row.ExperimentURL = mlflowExperimentURL(host, ids)
			// The ETA needs a progress-metric history fetch, so it's computed only
			// for running rows (the only ones that can have one).
			if entries[i].row.Status == string(jobs.RunLifeCycleStateRunning) {
				if eta := estimateTrainingETA(ctx, w, ids.RunID); eta != nil {
					entries[i].row.Progress = eta.compact()
				}
			}
			return nil
		})
	}
	// mlflowIDsForRetries never returns an error (it logs and yields nil), so Wait can't fail.
	_ = g.Wait()
}
