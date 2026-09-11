package aircmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

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
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 2,
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
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 1,
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
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 10,
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
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 10,
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
		{"active default → scan", listQuery{activeOnly: true, userFilter: "me@example.com", currentUser: "me@example.com"}, false},
		{"all-status self → index", listQuery{userFilter: "me@example.com", currentUser: "me@example.com", limit: 5}, true},
		{"all-status all-users → scan", listQuery{allUsers: true}, false},
		{"all-status other user → scan", listQuery{userFilter: "other@example.com", currentUser: "me@example.com"}, false},
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
	// Index 500 must silently fall back to the Jobs scan, not fail the command.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == aiTrainingWorkflowsPath {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"message":"boom"}`))
			return
		}
		if r.URL.Path == "/api/2.2/jobs/runs/list" {
			_, _ = fmt.Fprint(w, runsListBody(t, "", airBaseRun(1, "me@example.com", "GPU_1xH100", 1, "exp")))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("DATABRICKS_CACHE_ENABLED", "false")

	f := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 10,
	})
	rows, err := f.next(10)
	require.NoError(t, err)
	require.Len(t, rows, 1) // served by the Jobs scan fallback
}
