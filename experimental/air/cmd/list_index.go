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
// newest-first, keeps the newest `limit`, then hydrates them into full rows in
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

	ids    []int64 // newest-first run ids to hydrate, resolved on first next()
	pos    int
	loaded bool
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
	// Keep only the newest `limit` ids so hydration is bounded — but skip that when
	// a task filter is active, since it drops matches post-hydration and we'd
	// otherwise return fewer than `limit`. The caller stops pulling at `limit`.
	if s.limit > 0 && len(refs) > s.limit && !s.filters.hasTaskFilter() {
		refs = refs[:s.limit]
	}
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
	return entries, nil
}

func (s *indexStrategy) done() bool {
	return s.loaded && s.pos >= len(s.ids)
}

// truncated is always false: the index path is bounded by limit, not a scan cap.
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
		if row, fields, ok := cachedRow(s.ctx, s.cache, host, id); ok {
			if s.filters.matchesFields(fields) {
				rows = append(rows, listedRun{row: row, taskRunID: id})
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
		fields := filterFieldsFromRun(run)
		if !s.filters.matchesFields(fields) {
			continue
		}
		row := buildListRow(run, s.host, s.workspaceID)
		rows = append(rows, listedRun{row: row, taskRunID: taskRunID(run)})
		if isTerminal(run) {
			start, _ := jobTiming(run)
			putRow(s.ctx, s.cache, host, run.RunId, start, row, fields)
		}
	}

	// Concurrent hydration reorders runs, so re-sort the batch newest-first. The
	// ISO start timestamp sorts lexicographically; a missing time ("") sorts last.
	slices.SortStableFunc(rows, func(a, b listedRun) int {
		return cmp.Compare(rowStartKey(b.row), rowStartKey(a.row))
	})
	return rows, nil
}

// rowStartKey returns a row's ISO start timestamp for ordering, or "" when the
// run hasn't started (which sorts last under descending comparison).
func rowStartKey(r listRow) string {
	if r.StartedAt == nil {
		return ""
	}
	return *r.StartedAt
}
