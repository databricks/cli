package aircmd

import (
	"context"
	"net/http"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

// airRunTypeFilter is the ListRuns `filter` fragment that narrows to AI Runtime
// runs server-side, using the RunsListVisible index. It is a PUBLIC_UNDOCUMENTED
// query param not modeled by the typed SDK, so it is sent via the raw client.
// Gated by the jobs `enableAirRunTypeFilter` flag; when off, the server rejects
// it with INVALID_PARAMETER_VALUE ("not supported in this workspace") and the
// caller falls back to the client-side scan.
const airRunTypeFilter = "runType=AI_RUNTIME"

// filterUnsupportedMarker identifies the flag-off rejection so it can be told
// apart from a genuinely malformed filter (both are INVALID_PARAMETER_VALUE).
const filterUnsupportedMarker = "not supported in this workspace"

// indexRunStrategy lists AIR runs via the server-side AI-Runtime filter (the fast,
// complete, indexed path). On the first request it either succeeds — and pages
// the filtered runs/list — or, when the workspace has the filter flag off,
// transparently falls back to the client-side scan for the rest of the listing.
type indexRunStrategy struct {
	ctx        context.Context
	w          *databricks.WorkspaceClient
	api        *client.DatabricksClient
	activeOnly bool
	userFilter string
	filters    listFilters

	started   bool
	pageToken string
	drained   bool
	pending   []listedRun      // matched rows from a page beyond the last want
	fallback  *runScanStrategy // set when the filter is unsupported
}

func newIndexRunStrategy(ctx context.Context, w *databricks.WorkspaceClient, q listQuery) (*indexRunStrategy, error) {
	api, err := client.New(w.Config)
	if err != nil {
		return nil, err
	}
	return &indexRunStrategy{
		ctx:        ctx,
		w:          w,
		api:        api,
		activeOnly: q.activeOnly,
		userFilter: q.userFilter,
		filters:    q.filters,
	}, nil
}

// listRunsResponse is the runs/list shape the raw client unmarshals into. It
// mirrors jobs.ListRunsResponse (the typed SDK is bypassed only to pass the
// undocumented filter query param).
type listRunsResponse struct {
	Runs          []jobs.BaseRun `json:"runs"`
	NextPageToken string         `json:"next_page_token"`
}

func (s *indexRunStrategy) next(want int) ([]listedRun, error) {
	// A prior request found the filter unsupported: everything now goes through the
	// client-side scan.
	if s.fallback != nil {
		return s.fallback.next(want)
	}

	// Serve leftovers from a prior over-fetched page first.
	var entries []listedRun
	if len(s.pending) > 0 {
		n := min(want, len(s.pending))
		entries = append(entries, s.pending[:n]...)
		s.pending = s.pending[n:]
	}

	for len(entries) < want && !s.drained {
		page, err := s.fetchPage()
		if err != nil {
			// The workspace has the filter flag off: switch to the scan for the whole
			// listing. fetchPage only errs on the first request (before any run is
			// yielded), so nothing consumed is lost.
			if isFilterUnsupported(err) {
				log.Debugf(s.ctx, "air list: AI-Runtime filter unsupported, falling back to scan: %v", err)
				s.fallback = newRunScanStrategy(s.ctx, s.w, listQuery{
					activeOnly: s.activeOnly,
					userFilter: s.userFilter,
					filters:    s.filters,
				})
				return s.fallback.next(want)
			}
			return nil, err
		}
		if s.pageToken == "" {
			s.drained = true
		}

		for i := range page {
			run := baseRunToRun(page[i])
			// The server filter narrows to AIR runs; re-check by task shape as
			// insurance, and apply active-only, user-scoping, and task filters
			// client-side (active_only can't ride the filter query server-side).
			if !isAirRun(run) {
				continue
			}
			if s.activeOnly && !isActiveRun(run) {
				continue
			}
			if s.userFilter != "" && run.CreatorUserName != s.userFilter {
				continue
			}
			if !s.filters.matches(run) {
				continue
			}
			row := listedRun{row: buildListRow(run), taskRunID: taskRunID(run)}
			if len(entries) < want {
				entries = append(entries, row)
			} else {
				s.pending = append(s.pending, row) // carry to the next next() call
			}
		}
	}
	return entries, nil
}

// fetchPage issues one filtered runs/list request via the raw client and advances
// the page token.
func (s *indexRunStrategy) fetchPage() ([]jobs.BaseRun, error) {
	// active_only cannot be combined with the filter query server-side, so the
	// indexed path fetches all statuses and applies active-only filtering in next().
	query := map[string]any{
		"expand_tasks": true,
		"limit":        jobsPageLimit,
		"filter":       airRunTypeFilter,
	}
	if s.pageToken != "" {
		query["page_token"] = s.pageToken
	}
	var resp listRunsResponse
	err := s.api.Do(s.ctx, http.MethodGet, "/api/2.2/jobs/runs/list", nil, nil, query, &resp)
	if err != nil {
		return nil, err
	}
	s.started = true
	s.pageToken = resp.NextPageToken
	return resp.Runs, nil
}

func (s *indexRunStrategy) done() bool {
	if s.fallback != nil {
		return s.fallback.done()
	}
	return s.drained && len(s.pending) == 0
}

// truncated reports the scan cap only when the fallback is active; the indexed
// path is complete (no cap).
func (s *indexRunStrategy) truncated() bool {
	return s.fallback != nil && s.fallback.truncated()
}

// isFilterUnsupported reports whether err is the flag-off rejection of the
// AI-Runtime filter, distinguishing it from a genuinely bad filter value.
func isFilterUnsupported(err error) bool {
	return strings.Contains(err.Error(), filterUnsupportedMarker)
}
