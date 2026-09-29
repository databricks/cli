package aircmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCacheRoundTrip(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
	ctx := t.Context()
	c := newListCache(ctx)

	_, _, _, ok := cachedRow(ctx, c, "https://host.test", 42)
	require.False(t, ok, "miss before write")

	row := listRow{RunID: "42", Experiment: "exp", Status: "SUCCESS"}
	fields := filterFields{Experiment: "exp", GPUType: "GPU_1xA10", GPUCount: 1}
	putRow(ctx, c, "https://host.test", 42, 420, 1700000000000, row, fields)

	got, gotFields, taskRunID, ok := cachedRow(ctx, c, "https://host.test", 42)
	require.True(t, ok, "hit after write")
	assert.Equal(t, row, got)
	assert.Equal(t, fields, gotFields)
	assert.Equal(t, int64(420), taskRunID)

	// Different host is a different key.
	_, _, _, ok = cachedRow(ctx, c, "https://other.test", 42)
	assert.False(t, ok)

	// Entries from the previous cache format have no task run id and must be refreshed.
	putRow(ctx, c, "https://host.test", 43, 0, 1700000000000, listRow{RunID: "43"}, filterFields{})
	_, _, _, ok = cachedRow(ctx, c, "https://host.test", 43)
	assert.False(t, ok)
}

func TestIndexStrategyServesCachedRowWithoutFetch(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	refs := []workflowRef{{jobRunID: 7, submitTimeMs: 1000_000}}
	srv, hits := indexAndGetServer(t, refs, map[int64]jobs.Run{7: indexRun(7, 1000_000)}, nil, nil)
	host := srv.URL

	// Pre-seed the cache for run 7 so hydration should skip runs/get entirely.
	ctx := t.Context()
	putRow(ctx, newListCache(ctx), host, 7, 70, 1000_000, listRow{RunID: "7", Status: "SUCCESS"}, filterFields{})

	f := newRunFetcher(ctx, newTestWorkspaceClient(t, host), listQuery{
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 10,
	})
	rows, err := f.next(10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "7", rows[0].RunID)
	assert.Equal(t, 0, hits.get, "cached run must not hit runs/get")
}

func TestIndexStrategyFiltersCachedRow(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	refs := []workflowRef{{jobRunID: 7, submitTimeMs: 1000_000}}
	srv, hits := indexAndGetServer(t, refs, map[int64]jobs.Run{7: indexRun(7, 1000_000)}, nil, nil)
	host := srv.URL

	// Cache hit under experiment "bar" must be filtered out by an experiment=foo query.
	ctx := t.Context()
	putRow(ctx, newListCache(ctx), host, 7, 70, 1000_000,
		listRow{RunID: "7", Status: "SUCCESS", Experiment: "bar"},
		filterFields{Experiment: "bar"})

	f := newRunFetcher(ctx, newTestWorkspaceClient(t, host), listQuery{
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 10,
		filters: listFilters{Experiment: "foo"},
	})
	rows, err := f.next(10)
	require.NoError(t, err)
	assert.Empty(t, rows, "cached row not matching the filter must be dropped")
	assert.Equal(t, 0, hits.get, "non-matching cached row must not hit runs/get")
}

func TestIndexStrategyServesMatchingCachedRow(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	refs := []workflowRef{{jobRunID: 7, submitTimeMs: 1000_000}}
	srv, hits := indexAndGetServer(t, refs, map[int64]jobs.Run{7: indexRun(7, 1000_000)}, nil, nil)
	host := srv.URL

	ctx := t.Context()
	putRow(ctx, newListCache(ctx), host, 7, 70, 1000_000,
		listRow{RunID: "7", Status: "SUCCESS", Experiment: "foo"},
		filterFields{Experiment: "foo"})

	f := newRunFetcher(ctx, newTestWorkspaceClient(t, host), listQuery{
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 10,
		filters: listFilters{Experiment: "foo"},
	})
	rows, err := f.next(10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "7", rows[0].RunID)
	assert.Equal(t, 0, hits.get, "matching cached row must not hit runs/get")
}

func TestIndexStrategyCachesMLflowEnrichment(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	run := indexRun(7, 1000_000)
	run.Tasks[0].RunId = 70
	var getHits, outputHits, mlflowHits int
	var outputRunIDs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case aiTrainingWorkflowsPath:
			_ = json.NewEncoder(w).Encode(map[string]any{"training_workflows": []map[string]any{{
				"job_run_id": "7", "submit_time": map[string]any{"seconds": 1000},
			}}})
		case "/api/2.2/jobs/runs/get":
			getHits++
			_ = json.NewEncoder(w).Encode(run)
		case "/api/2.2/jobs/runs/get-output":
			outputHits++
			outputRunIDs = append(outputRunIDs, r.URL.Query().Get("run_id"))
			if r.URL.Query().Get("run_id") != "70" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"message":"get-output requires a task run id"}`))
				return
			}
			_, _ = w.Write([]byte(`{"ai_runtime_task_output":{"mlflow_experiment_id":"exp1","mlflow_run_id":"mlflow1"}}`))
		case "/api/2.0/mlflow/runs/get":
			mlflowHits++
			_, _ = w.Write([]byte(`{"run":{"info":{"run_name":"training-run"}}}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)

	query := listQuery{
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 10,
	}
	rows, err := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), query).next(10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "-", rows[0].MLflowLabel)

	query.fetchMLflow = true
	for range 2 {
		rows, err = newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), query).next(10)
		require.NoError(t, err)
		require.Len(t, rows, 1)
		assert.Equal(t, "training-run", rows[0].MLflowLabel)
	}

	assert.Equal(t, 1, getHits, "warm cache must skip runs/get")
	assert.Equal(t, 1, outputHits, "warm cache must skip get-output")
	assert.Equal(t, 1, mlflowHits, "warm cache must skip the MLflow lookup")
	assert.Equal(t, []string{"70"}, outputRunIDs, "get-output must use the task run id")
}

func TestIsTerminal(t *testing.T) {
	assert.True(t, isTerminal(&jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated}}))
	assert.True(t, isTerminal(&jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateInternalError}}))
	assert.False(t, isTerminal(&jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning}}))
	assert.False(t, isTerminal(&jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStatePending}}))
}
