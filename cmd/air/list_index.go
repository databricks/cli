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

func (s *indexStrategy) next(want int) ([]runGroup, error) {
	if !s.loaded {
		if err := s.load(); err != nil {
			return nil, err
		}
	}
	if s.limit > 0 {
		want = min(want, s.limit-s.returned)
	}

	// want counts runs, as on the scan path: a run with several AI runtime
	// tasks hydrates into one group of rows.
	var groups []runGroup
	for len(groups) < want && s.pos < len(s.ids) {
		end := min(s.pos+want-len(groups), len(s.ids))
		batch := s.ids[s.pos:end]
		s.pos = end

		hydrated, err := s.hydrate(batch)
		if err != nil {
			return nil, err
		}
		groups = append(groups, hydrated...)
	}
	s.returned += len(groups)
	return groups, nil
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
func (s *indexStrategy) hydrate(ids []int64) ([]runGroup, error) {
	host := s.w.Config.Host

	groups := make([]runGroup, 0, len(ids))
	var toFetch []int64
	// The cache key excludes the filter, so a cached run is still run through the
	// active filter; a non-matching hit is dropped, not re-fetched.
	for _, id := range ids {
		if entries, ok := cachedEntries(s.ctx, s.cache, host, id); ok {
			if g, ok := matchingGroup(id, entries, s.filters); ok {
				groups = append(groups, g)
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
		entries := buildTaskEntries(run, s.host, s.workspaceID)
		if isTerminal(run) {
			start, _ := jobTiming(defaultTaskRun(run))
			putEntries(s.ctx, s.cache, host, run.RunId, entries, start)
		}
		if g, ok := matchingGroup(run.RunId, entries, s.filters); ok {
			groups = append(groups, g)
		}
	}

	// Concurrent hydration reorders runs, so re-sort the batch newest-first,
	// keeping a run's rows together. The ISO start timestamp sorts
	// lexicographically; a missing time ("") sorts last.
	slices.SortStableFunc(groups, func(a, b runGroup) int {
		return cmp.Compare(groupStartKey(b), groupStartKey(a))
	})
	return groups, nil
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

// groupStartKey returns the start key of a run's first row that has started: a
// task that hasn't started or was skipped has none (see taskRun), and shouldn't
// sort its whole run last.
func groupStartKey(g runGroup) string {
	for _, e := range g.entries {
		if k := rowStartKey(e.row); k != "" {
			return k
		}
	}
	return ""
}

// rowStartKey returns a row's ISO start timestamp for ordering, or "" when the
// run hasn't started (which sorts last under descending comparison).
func rowStartKey(r listRow) string {
	if r.StartedAt == nil {
		return ""
	}
	return *r.StartedAt
}
