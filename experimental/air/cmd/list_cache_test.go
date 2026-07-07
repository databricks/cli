package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCacheRoundTrip(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
	ctx := t.Context()
	c := newListCache(ctx)

	_, ok := cachedRow(ctx, c, "https://host.test", 42)
	require.False(t, ok, "miss before write")

	row := listRow{RunID: "42", Experiment: "exp", Status: "SUCCESS"}
	putRow(ctx, c, "https://host.test", 42, 1700000000000, row)

	got, ok := cachedRow(ctx, c, "https://host.test", 42)
	require.True(t, ok, "hit after write")
	assert.Equal(t, row, got)

	// Different host is a different key.
	_, ok = cachedRow(ctx, c, "https://other.test", 42)
	assert.False(t, ok)
}

func TestIndexStrategyServesCachedRowWithoutFetch(t *testing.T) {
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())

	refs := []workflowRef{{jobRunID: 7, submitTimeMs: 1000_000}}
	srv, hits := indexAndGetServer(t, refs, map[int64]jobRun{7: indexRun(7, 1000_000)}, nil, nil)
	host := srv.URL

	// Pre-seed the cache for run 7 so hydration should skip runs/get entirely.
	ctx := t.Context()
	putRow(ctx, newListCache(ctx), host, 7, 1000_000, listRow{RunID: "7", Status: "SUCCESS"})

	f := newRunFetcher(ctx, newTestWorkspaceClient(t, host), listQuery{
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 10,
	})
	rows, err := f.next(10)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "7", rows[0].RunID)
	assert.Equal(t, 0, hits.get, "cached run must not hit runs/get")
}

func TestIsTerminal(t *testing.T) {
	assert.True(t, isTerminal(&jobRun{State: jobState{LifeCycleState: "TERMINATED"}}))
	assert.True(t, isTerminal(&jobRun{State: jobState{LifeCycleState: "INTERNAL_ERROR"}}))
	assert.False(t, isTerminal(&jobRun{State: jobState{LifeCycleState: "RUNNING"}}))
	assert.False(t, isTerminal(&jobRun{State: jobState{LifeCycleState: "PENDING"}}))
}
