package aircmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// filterCapturingServer records the `filter` query param on runs/list and serves
// the given body; it lets a test assert the AI-Runtime filter is sent.
func filterCapturingServer(t *testing.T, gotFilter *string, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/list" {
			*gotFilter = r.URL.Query().Get("filter")
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestIndexStrategySendsAirFilter(t *testing.T) {
	var filter string
	run := airBaseRun(9, "me@example.com", "GPU_1xA10", 1, "exp")
	srv := filterCapturingServer(t, &filter, runsListBody(t, "", run))

	rows, err := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		activeOnly: true, userFilter: "me@example.com",
	}).next(10)
	require.NoError(t, err)
	assert.Equal(t, "runType=AI_RUNTIME", filter, "list must request the server-side AI-Runtime filter")
	require.Len(t, rows, 1)
	assert.Equal(t, "9", rows[0].RunID)
}

// paramCapturingServer records the filter and active_only query params on runs/list.
func paramCapturingServer(t *testing.T, gotFilter, gotActiveOnly *string, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/list" {
			*gotFilter = r.URL.Query().Get("filter")
			*gotActiveOnly = r.URL.Query().Get("active_only")
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestIndexStrategyActiveOnlyFiltersClientSide(t *testing.T) {
	// active_only cannot ride the filter query server-side, so the indexed request
	// must NOT send active_only; active filtering happens client-side.
	running := airBaseRun(1, "me@example.com", "GPU_1xA10", 1, "exp-a")
	terminated := airBaseRun(2, "me@example.com", "GPU_1xA10", 1, "exp-b")
	terminated.State = &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated, ResultState: jobs.RunResultStateSuccess}

	var filter, activeOnly string
	srv := paramCapturingServer(t, &filter, &activeOnly, runsListBody(t, "", running, terminated))

	rows, err := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		activeOnly: true, userFilter: "me@example.com",
	}).next(10)
	require.NoError(t, err)
	assert.Equal(t, "runType=AI_RUNTIME", filter)
	assert.Empty(t, activeOnly, "active_only must not be sent alongside the filter query")
	require.Len(t, rows, 1, "the terminated run is filtered out client-side")
	assert.Equal(t, "1", rows[0].RunID)
}

// unsupportedFilterServer rejects the AI-Runtime filter like a flag-off
// workspace (400 INVALID_PARAMETER_VALUE, "not supported in this workspace"),
// so a test can exercise the fallback to the client-side scan.
func unsupportedFilterServer(t *testing.T, scanBody string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/list" {
			if r.URL.Query().Get("filter") != "" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error_code":"INVALID_PARAMETER_VALUE","message":"runType=AI_RUNTIME filter is not supported in this workspace"}`))
				return
			}
			_, _ = w.Write([]byte(scanBody))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestIndexStrategyFallsBackWhenUnsupported(t *testing.T) {
	// The filtered request is rejected (flag off); the scan (no filter) then serves
	// the runs.
	scan := runsListBody(t, "",
		airBaseRun(1, "me@example.com", "GPU_1xA10", 1, "exp-a"),
		airBaseRun(2, "me@example.com", "GPU_1xA10", 1, "exp-b"),
	)
	srv := unsupportedFilterServer(t, scan)

	rows, err := newRunFetcher(t.Context(), newTestWorkspaceClient(t, srv.URL), listQuery{
		activeOnly: true, userFilter: "me@example.com",
	}).next(10)
	require.NoError(t, err)
	require.Len(t, rows, 2, "falls back to the scan and returns its runs")
	assert.Equal(t, "1", rows[0].RunID)
	assert.Equal(t, "2", rows[1].RunID)
}
