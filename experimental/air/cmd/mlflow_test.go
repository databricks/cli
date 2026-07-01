package aircmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestWorkspaceClient builds a WorkspaceClient pointed at a mock HTTP server.
// mlflowIDs calls the runs/get-output REST endpoint directly (the field it needs
// is not modeled by the typed SDK), so it must be exercised over HTTP.
func newTestWorkspaceClient(t *testing.T, host string) *databricks.WorkspaceClient {
	t.Helper()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: host, Token: "token"})
	require.NoError(t, err)
	return w
}

// runOutputServer serves the given runs/get-output body and a stub for the SDK's
// well-known config discovery request. *hit is set when get-output is called.
func runOutputServer(t *testing.T, body string, hit *bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/get-output" {
			*hit = true
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestMLflowIDs(t *testing.T) {
	ctx := t.Context()
	run := &jobs.Run{Tasks: []jobs.RunTask{{RunId: 99}}}

	t.Run("returns the identifiers on success", func(t *testing.T) {
		var hit bool
		srv := runOutputServer(t, `{"gen_ai_compute_output":{"run_info":{"mlflow_experiment_id":"E1","mlflow_run_id":"R1"}}}`, &hit)

		got := mlflowIDs(ctx, newTestWorkspaceClient(t, srv.URL), run)
		require.NotNil(t, got)
		assert.True(t, hit, "runs/get-output should have been called")
		assert.Equal(t, &mlflowIdentifiers{ExperimentID: "E1", RunID: "R1"}, got)
	})

	t.Run("nil when the run has no MLflow info", func(t *testing.T) {
		var hit bool
		srv := runOutputServer(t, `{}`, &hit)
		assert.Nil(t, mlflowIDs(ctx, newTestWorkspaceClient(t, srv.URL), run))
	})

	t.Run("nil when the run has no tasks", func(t *testing.T) {
		// Returns before any HTTP call, so the host is never contacted.
		assert.Nil(t, mlflowIDs(ctx, newTestWorkspaceClient(t, "https://unused.invalid"), &jobs.Run{}))
	})

	t.Run("uses the latest attempt's task run", func(t *testing.T) {
		// A retried run must link to the last task, not the stale first attempt.
		var gotRunID string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/2.2/jobs/runs/get-output" {
				gotRunID = r.URL.Query().Get("run_id")
			}
			_, _ = w.Write([]byte(`{}`))
		}))
		t.Cleanup(srv.Close)

		retried := &jobs.Run{Tasks: []jobs.RunTask{{RunId: 99}, {RunId: 100}}}
		mlflowIDs(ctx, newTestWorkspaceClient(t, srv.URL), retried)
		assert.Equal(t, "100", gotRunID)
	})
}

func TestMLflowIDsForTask(t *testing.T) {
	ctx := t.Context()

	t.Run("parses ai_runtime_task_output", func(t *testing.T) {
		var hit bool
		srv := runOutputServer(t, `{"ai_runtime_task_output":{"mlflow_experiment_id":"E1","mlflow_run_id":"R1"}}`, &hit)
		got := mlflowIDsForTask(ctx, newTestWorkspaceClient(t, srv.URL), 99)
		require.NotNil(t, got)
		assert.True(t, hit)
		assert.Equal(t, &mlflowIdentifiers{ExperimentID: "E1", RunID: "R1"}, got)
	})

	t.Run("parses legacy gen_ai_compute_output", func(t *testing.T) {
		var hit bool
		srv := runOutputServer(t, `{"gen_ai_compute_output":{"run_info":{"mlflow_experiment_id":"E2","mlflow_run_id":"R2"}}}`, &hit)
		got := mlflowIDsForTask(ctx, newTestWorkspaceClient(t, srv.URL), 99)
		require.NotNil(t, got)
		assert.Equal(t, &mlflowIdentifiers{ExperimentID: "E2", RunID: "R2"}, got)
	})

	t.Run("nil when no task run id", func(t *testing.T) {
		// Returns before any HTTP call, so the host is never contacted.
		assert.Nil(t, mlflowIDsForTask(ctx, newTestWorkspaceClient(t, "https://unused.invalid"), 0))
	})
}

func TestMLflowURLs(t *testing.T) {
	ids := &mlflowIdentifiers{ExperimentID: "E1", RunID: "R1"}
	// A trailing slash on the host must not produce a double slash in the link.
	assert.Equal(t, "https://h.test/ml/experiments/E1/runs/R1/artifacts/logs/node_0", mlflowLogsURL("https://h.test/", ids))
	assert.Equal(t, "https://h.test/ml/experiments/E1/runs/R1", mlflowRunURL("https://h.test", ids))
}

func TestMLflowRunLabel(t *testing.T) {
	// Uses the run name when it is known.
	assert.Equal(t, "sunny-cat-42", mlflowRunLabel("sunny-cat-42", "0123456789abcdef"))
	// Falls back to the last 8 characters of a long run id.
	assert.Equal(t, "...9abcdef0", mlflowRunLabel("", "0123456789abcdef0"))
	// A short run id is shown in full behind the ellipsis.
	assert.Equal(t, "...short", mlflowRunLabel("", "short"))
}

func TestFetchMLflowRunName(t *testing.T) {
	ctx := t.Context()

	mlflowServer := func(body string) *httptest.Server {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/2.0/mlflow/runs/get" {
				_, _ = w.Write([]byte(body))
				return
			}
			_, _ = w.Write([]byte(`{}`))
		}))
		t.Cleanup(srv.Close)
		return srv
	}

	t.Run("returns the run name", func(t *testing.T) {
		srv := mlflowServer(`{"run":{"info":{"run_name":"sunny-cat-42"}}}`)
		assert.Equal(t, "sunny-cat-42", fetchMLflowRunName(ctx, newTestWorkspaceClient(t, srv.URL), "run1"))
	})

	t.Run("empty when the run cannot be fetched", func(t *testing.T) {
		srv := mlflowServer(`{}`)
		assert.Empty(t, fetchMLflowRunName(ctx, newTestWorkspaceClient(t, srv.URL), "run1"))
	})
}
