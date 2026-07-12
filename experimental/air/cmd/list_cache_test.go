package aircmd

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCacheRoundTrip(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
	ctx := t.Context()
	c := newListCache(ctx)

	_, _, ok := cachedRow(ctx, c, "https://host.test", 42)
	require.False(t, ok, "miss before write")

	row := listRow{RunID: "42", Experiment: "exp", Status: "SUCCESS"}
	fields := filterFields{Experiment: "exp", GPUType: "GPU_1xA10", GPUCount: 1}
	putRow(ctx, c, "https://host.test", 42, 1700000000000, row, fields)

	got, gotFields, ok := cachedRow(ctx, c, "https://host.test", 42)
	require.True(t, ok, "hit after write")
	assert.Equal(t, row, got)
	assert.Equal(t, fields, gotFields)

	// Different host is a different key.
	_, _, ok = cachedRow(ctx, c, "https://other.test", 42)
	assert.False(t, ok)
}

func TestIndexStrategyServesCachedRowWithoutFetch(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	refs := []workflowRef{{jobRunID: 7, submitTimeMs: 1000_000}}
	srv, hits := indexAndGetServer(t, refs, map[int64]jobs.Run{7: indexRun(7, 1000_000)}, nil, nil)
	host := srv.URL

	// Pre-seed the cache for run 7 so hydration should skip runs/get entirely.
	ctx := t.Context()
	putRow(ctx, newListCache(ctx), host, 7, 1000_000, listRow{RunID: "7", Status: "SUCCESS"}, filterFields{})

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
	putRow(ctx, newListCache(ctx), host, 7, 1000_000,
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
	putRow(ctx, newListCache(ctx), host, 7, 1000_000,
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

func TestIsTerminal(t *testing.T) {
	assert.True(t, isTerminal(&jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated}}))
	assert.True(t, isTerminal(&jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateInternalError}}))
	assert.False(t, isTerminal(&jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning}}))
	assert.False(t, isTerminal(&jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStatePending}}))
}
