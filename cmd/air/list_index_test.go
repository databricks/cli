package aircmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// indexRun is a terminal AIR run with a start time, for index-path hydration.
func indexRun(id, startMillis int64) jobs.Run {
	r := airRun(id, "me@example.com", "GPU_1xH100", 1, "/Users/me@example.com/exp")
	r.State = &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated, ResultState: jobs.RunResultStateSuccess}
	r.Tasks[0].StartTime = startMillis
	r.Tasks[0].EndTime = startMillis + 1000
	return r
}

// indexAndGetServer serves the AiTrainingService index (a single page of the
// given refs) and runs/get for each id, recording hit counts per endpoint. A
// runID in forbidden returns 403; in missing returns 404.
type indexHits struct{ index, get int }

func indexAndGetServer(t *testing.T, refs []workflowRef, runs map[int64]jobs.Run, forbidden, missing map[int64]bool) (*httptest.Server, *indexHits) {
	t.Helper()
	hits := &indexHits{}
	wfs := make([]map[string]any, len(refs))
	for i, r := range refs {
		wfs[i] = map[string]any{"job_run_id": strconv.FormatInt(r.jobRunID, 10), "submit_time": map[string]any{"seconds": r.submitTimeMs / 1000}}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case aiTrainingWorkflowsPath:
			hits.index++
			_ = json.NewEncoder(w).Encode(map[string]any{"training_workflows": wfs})
		case "/api/2.2/jobs/runs/get":
			hits.get++
			id, _ := strconv.ParseInt(r.URL.Query().Get("run_id"), 10, 64)
			if forbidden[id] {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"message":"forbidden"}`))
				return
			}
			if missing[id] {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"message":"not found"}`))
				return
			}
			run := runs[id]
			_ = json.NewEncoder(w).Encode(run)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, hits
}

// selfUser resolves the current user for a self-scoped listQuery.
func selfUser() (string, error) { return "me@example.com", nil }

func TestIndexStrategyOrdersAndLimits(t *testing.T) {
	// Three runs, out of submit-time order; newest two should win, newest-first.
	refs := []workflowRef{
		{jobRunID: 1, submitTimeMs: 1000_000},
		{jobRunID: 2, submitTimeMs: 3000_000},
		{jobRunID: 3, submitTimeMs: 2000_000},
	}
	runs := map[int64]jobs.Run{
		1: indexRun(1, 1000_000),
		2: indexRun(2, 3000_000),
		3: indexRun(3, 2000_000),
	}
	srv, _ := indexAndGetServer(t, refs, runs, nil, nil)
	t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

	f := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		currentUser: selfUser, limit: 2,
	})
	rows, err := f.next(10)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "2", rows[0].RunID) // submit 3000
	assert.Equal(t, "3", rows[1].RunID) // submit 2000
	assert.True(t, f.exhausted)
}

func TestIndexStrategyOverFetchesWithTaskFilter(t *testing.T) {
	// With a task filter and limit 1, the newest run doesn't match; the strategy
	// must keep hydrating past `limit` to find the match rather than truncating.
	refs := []workflowRef{
		{jobRunID: 1, submitTimeMs: 3000_000},
		{jobRunID: 2, submitTimeMs: 2000_000},
	}
	run1 := indexRun(1, 3000_000)
	run1.Tasks[0].AiRuntimeTask.Experiment = "/Users/me@example.com/llama"
	run2 := indexRun(2, 2000_000)
	run2.Tasks[0].AiRuntimeTask.Experiment = "/Users/me@example.com/qwen"
	srv, _ := indexAndGetServer(t, refs, map[int64]jobs.Run{1: run1, 2: run2}, nil, nil)
	t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

	f := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		currentUser: selfUser, limit: 1,
		filters: listFilters{Experiment: "qwen"},
	})
	rows, err := f.next(1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "2", rows[0].RunID) // found despite being the older, second id
}

func TestIndexStrategyDropsForbiddenAndMissing(t *testing.T) {
	refs := []workflowRef{
		{jobRunID: 1, submitTimeMs: 3000_000},
		{jobRunID: 2, submitTimeMs: 2000_000},
		{jobRunID: 3, submitTimeMs: 1000_000},
	}
	runs := map[int64]jobs.Run{1: indexRun(1, 3000_000), 3: indexRun(3, 1000_000)}
	srv, _ := indexAndGetServer(t, refs, runs, map[int64]bool{2: true}, nil)
	t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

	f := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		currentUser: selfUser, limit: 10,
	})
	rows, err := f.next(10)
	require.NoError(t, err)
	require.Len(t, rows, 2) // run 2 (403) dropped
	assert.Equal(t, "1", rows[0].RunID)
	assert.Equal(t, "3", rows[1].RunID)
}

func TestIndexStrategyPropagatesServerError(t *testing.T) {
	refs := []workflowRef{{jobRunID: 1, submitTimeMs: 1000_000}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case aiTrainingWorkflowsPath:
			wfs := []map[string]any{{"job_run_id": "1", "submit_time": map[string]any{"seconds": int64(1000)}}}
			_ = json.NewEncoder(w).Encode(map[string]any{"training_workflows": wfs})
		case "/api/2.2/jobs/runs/get":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"boom"}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	_ = refs
	t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

	f := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		currentUser: selfUser, limit: 10,
	})
	_, err := f.next(10)
	require.Error(t, err) // 500 is systemic, not an ACL drop
}

func TestNewListStrategyGate(t *testing.T) {
	// --all-users and other-user filters must NOT touch the index endpoint.
	cases := []struct {
		name      string
		q         listQuery
		wantIndex bool
	}{
		{"active default → scan", listQuery{activeOnly: true, currentUser: selfUser}, false},
		{"all-status self → index", listQuery{currentUser: selfUser, limit: 5}, true},
		{"all-status all-users → scan", listQuery{allUsers: true}, false},
		{"all-status other user → scan", listQuery{userFilter: "other@example.com"}, false},
		{"all-status self filter, other case → index", listQuery{userFilter: "Me@Example.com", selfUser: "me@example.com", limit: 5}, true},
		{"all-status filter, current user unknown → scan", listQuery{userFilter: "me@example.com"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var indexHit bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == aiTrainingWorkflowsPath {
					indexHit = true
				}
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(srv.Close)
			t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

			newListStrategy(t.Context(), newTestWorkspaceClient(t, srv.URL), tc.q)
			assert.Equal(t, tc.wantIndex, indexHit)
		})
	}
}

func TestNewListStrategyFallsBackWhenIndexFails(t *testing.T) {
	// Index 500 must fall back to the Jobs scan, not fail the command.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == aiTrainingWorkflowsPath {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"boom"}`))
			return
		}
		if r.URL.Path == "/api/2.2/jobs/runs/list" {
			_, _ = fmt.Fprint(w, runsListBody(t, "",
				airBaseRun(1, "me@example.com", "GPU_1xH100", 1, "exp"),
				airBaseRun(2, "other@example.com", "GPU_1xH100", 1, "exp")))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

	f := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		currentUser: selfUser, limit: 10,
	})
	rows, err := f.next(10)
	require.NoError(t, err)
	// Served by the Jobs scan fallback, still scoped to the current user.
	require.Len(t, rows, 1)
	assert.Equal(t, "1", rows[0].RunID)
}

func TestIndexStrategySkipsUserLookup(t *testing.T) {
	refs := []workflowRef{{jobRunID: 1, submitTimeMs: 1000_000}}
	srv, _ := indexAndGetServer(t, refs, map[int64]jobs.Run{1: indexRun(1, 1000_000)}, nil, nil)
	t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

	// The index is already scoped to the caller, so it never needs the user name.
	var lookups atomic.Int32
	f := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		currentUser: func() (string, error) { lookups.Add(1); return "me@example.com", nil },
		limit:       10,
	})
	rows, err := f.next(10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Zero(t, lookups.Load())
}

// captureWarnings routes warnings logged through ctx into a buffer.
func captureWarnings(ctx context.Context) (context.Context, *bytes.Buffer) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	return log.NewContext(ctx, logger), &buf
}

// endlessNonAirRunsPage is a full runs/list page of non-AIR runs that always
// points at another page, standing in for a deep run history.
func endlessNonAirRunsPage(t *testing.T) string {
	runs := make([]jobs.BaseRun, jobsPageLimit)
	for i := range runs {
		runs[i] = jobs.BaseRun{RunId: int64(i + 1), CreatorUserName: "me@example.com", Tasks: []jobs.RunTask{{}}}
	}
	return runsListBody(t, "more", runs...)
}

func TestFallbackScanWarnsAndIsBounded(t *testing.T) {
	var listCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case aiTrainingWorkflowsPath:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"boom"}`))
		case "/api/2.2/jobs/runs/list":
			listCalls.Add(1)
			assert.Empty(t, r.URL.Query().Get("active_only"))
			_, _ = fmt.Fprint(w, endlessNonAirRunsPage(t))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

	var stdout bytes.Buffer
	ctx, warnings := captureWarnings(t.Context())
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &stdout, &stdout, "", ""))
	f := newRunFetcher(ctx, newTestWorkspaceClient(t, srv.URL), listQuery{
		userFilter: "me@example.com", selfUser: "me@example.com", limit: 10,
	})
	rows, err := f.next(10)
	require.NoError(t, err)
	assert.Empty(t, rows)
	warnIfTruncated(ctx, f)

	// The fallback stops at its own cap instead of maxListScan.
	assert.Equal(t, int32(maxFallbackListScan/jobsPageLimit), listCalls.Load())
	assert.Contains(t, warnings.String(), "run index is unavailable")
	assert.Contains(t, warnings.String(), "falling back to a slower scan of up to 250 recent job runs")
	assert.Contains(t, warnings.String(), "stopped after scanning 250 runs; results may be incomplete")
	assert.Empty(t, stdout.String())
}

func TestRegularScanKeepsFullCap(t *testing.T) {
	var listCalls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/list" {
			listCalls.Add(1)
			_, _ = fmt.Fprint(w, endlessNonAirRunsPage(t))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	ctx, warnings := captureWarnings(t.Context())
	f := newRunFetcher(ctx, newTestWorkspaceClient(t, srv.URL), listQuery{allUsers: true, limit: 10})
	_, err := f.next(10)
	require.NoError(t, err)
	warnIfTruncated(ctx, f)

	assert.Equal(t, int32(maxListScan/jobsPageLimit), listCalls.Load())
	assert.NotContains(t, warnings.String(), "run index is unavailable")
	assert.Contains(t, warnings.String(), "stopped after scanning 2000 runs; results may be incomplete")
}

// TestListSelfUserFilterUsesIndex pins that --all-status --filter user=<own
// email> is served by the index, like the unfiltered self list, rather than a
// full-history Jobs scan.
func TestListSelfUserFilterUsesIndex(t *testing.T) {
	var indexHits, listHits atomic.Int32
	run := indexRun(7, 3000_000)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/2.0/preview/scim/v2/Me":
			_, _ = w.Write([]byte(`{"userName":"me@example.com"}`))
		case aiTrainingWorkflowsPath:
			indexHits.Add(1)
			_, _ = w.Write([]byte(`{"training_workflows":[{"job_run_id":"7","submit_time":{"seconds":3000}}]}`))
		case "/api/2.2/jobs/runs/get":
			_ = json.NewEncoder(w).Encode(run)
		case "/api/2.2/jobs/runs/list":
			listHits.Add(1)
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

	var stdout bytes.Buffer
	ctx, warnings := captureWarnings(t.Context())
	ctx = cmdctx.SetWorkspaceClient(ctx, newTestWorkspaceClient(t, srv.URL))
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputJSON, nil, &stdout, &stdout, "", ""))
	cmd := withOutput(newListCommand(), flags.OutputJSON)
	cmd.SetContext(ctx)
	require.NoError(t, cmd.Flags().Set("all-status", "true"))
	require.NoError(t, cmd.Flags().Set("filter", "user=me@example.com"))

	require.NoError(t, cmd.RunE(cmd, nil))
	assert.Equal(t, int32(1), indexHits.Load())
	assert.Equal(t, int32(0), listHits.Load())
	assert.Contains(t, stdout.String(), `"run_id": "7"`)
	assert.Empty(t, warnings.String())
}
