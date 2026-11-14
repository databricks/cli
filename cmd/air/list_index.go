package aircmd

import (
	"cmp"
	"context"
	"slices"

	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/databricks-sdk-go"
)

// indexStrategy serves the caller's own runs from the AiTrainingService index:
// it fetches every run id up front (cheap id+timestamp pairs), orders them
// newest-first, then hydrates them into full rows in
// want-sized batches via Jobs runs/get. Terminal rows are cached so repeat calls
// skip the network. Unlike the Jobs scan it can't lazy-page (it must sort the
// whole id set first), but it still yields in batches so the table paints early.
type indexStrategy struct {
	ctx         context.Context
	w           *databricks.WorkspaceClient
	activeOnly  bool
	filters     listFilters
	limit       int
	cache       *cache.Cache
	host        string
	workspaceID int64

	ids      []int64 // newest-first run ids to hydrate, resolved on first next()
	pos      int
	returned int
	loaded   bool
}

func newIndexStrategy(ctx context.Context, w *databricks.WorkspaceClient, q listQuery, limit int) *indexStrategy {
	return &indexStrategy{
		ctx:         ctx,
		w:           w,
		activeOnly:  q.activeOnly,
		filters:     q.filters,
		limit:       limit,
		cache:       newListCache(ctx),
		host:        w.Config.Host,
		workspaceID: q.workspaceID,
	}
}

// load fetches and orders the index once. It returns an error only when the
// index endpoint itself fails, letting the caller fall back to the Jobs scan.
func (s *indexStrategy) load() error {
	refs, err := listAiTrainingWorkflows(s.ctx, s.w, s.activeOnly)
	if err != nil {
		return err
	}
	slices.SortFunc(refs, func(a, b workflowRef) int { return cmp.Compare(b.submitTimeMs, a.submitTimeMs) })
	s.ids = make([]int64, len(refs))
	for i, r := range refs {
		s.ids[i] = r.jobRunID
	}
	s.loaded = true
	return nil
}

func (s *indexStrategy) next(want int) ([]listedRun, error) {
	if !s.loaded {
		if err := s.load(); err != nil {
			return nil, err
		}
	}
	if s.limit > 0 {
		want = min(want, s.limit-s.returned)
	}

	var entries []listedRun
	for len(entries) < want && s.pos < len(s.ids) {
		end := min(s.pos+want-len(entries), len(s.ids))
		batch := s.ids[s.pos:end]
		s.pos = end

		rows, err := s.hydrate(batch)
		if err != nil {
			return nil, err
		}
		entries = append(entries, rows...)
	}
	s.returned += len(entries)
	return entries, nil
}

func (s *indexStrategy) done() bool {
	return s.loaded && ((s.limit > 0 && s.returned >= s.limit) || s.pos >= len(s.ids))
}

// truncated is always false: the index path stops at the requested limit, not a scan cap.
func (s *indexStrategy) truncated() bool { return false }

// hydrate turns a batch of run ids into rows, serving cached terminal rows
// without a network call and fetching the rest via runs/get. Freshly hydrated
// terminal runs are cached. Results keep the input (newest-first) order, then
// the batch is re-sorted by start time since concurrent hydration reorders it.
func (s *indexStrategy) hydrate(ids []int64) ([]listedRun, error) {
	host := s.w.Config.Host

	rows := make([]listedRun, 0, len(ids))
	var toFetch []int64
	// The cache key excludes the filter, so a cached row is still run through the
	// active filter; a non-matching hit is dropped, not re-fetched.
	for _, id := range ids {
		if cached, ok := cachedRows(s.ctx, s.cache, host, id); ok {
			for _, c := range cached {
				if s.filters.matchesFields(c.Fields) {
					rows = append(rows, listedRun{row: c.toRow(), runID: id, taskKey: c.TaskKey, tasks: c.Tasks, terminal: true})
				}
			}
			continue
		}
		toFetch = append(toFetch, id)
	}

	runs, err := hydrateJobRuns(s.ctx, s.w, toFetch)
	if err != nil {
		return nil, err
	}
	for _, run := range runs {
		// One row per AI runtime task; the whole set is cached so a later lookup
		// by run id can re-filter every task.
		entries := buildListRows(run, s.host, s.workspaceID)
		start, _ := jobTiming(run)
		cached := make([]cachedRun, 0, len(entries))
		for _, e := range entries {
			cached = append(cached, cachedRunFromRow(e.entry.row, e.entry.tasks, e.fields, start))
			if s.filters.matchesFields(e.fields) {
				rows = append(rows, e.entry)
			}
		}
		if isTerminal(run) {
			putRows(s.ctx, s.cache, host, run.RunId, cached)
		}
	}

	// Concurrent hydration reorders runs, so re-sort the batch newest-first. The
	// ISO start timestamp sorts lexicographically; a missing time ("") sorts last.
	slices.SortStableFunc(rows, func(a, b listedRun) int {
		return cmp.Compare(rowStartKey(b.row), rowStartKey(a.row))
	})
	return rows, nil
}

// cacheMLflowLinks writes successful MLflow enrichment back to terminal rows
// already in the cache. Active rows are absent from the cache and are ignored.
func (s *indexStrategy) cacheMLflowLinks(entries []listedRun) {
	for _, entry := range entries {
		if entry.row.MLflowURL != "" && entry.row.MLflowURL != "-" {
			updateCachedRow(s.ctx, s.cache, s.host, entry.runID, entry.taskKey, entry.row)
		}
	}
}

// rowStartKey returns a row's ISO start timestamp for ordering, or "" when the
// run hasn't started (which sorts last under descending comparison).
func rowStartKey(r listRow) string {
	if r.StartedAt == nil {
		return ""
	}
	return *r.StartedAt
}
