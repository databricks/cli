package aircmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestWorkspaceClient builds a WorkspaceClient pointed at a mock HTTP server,
// so the SDK calls these tests exercise are served locally.
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
		srv := runOutputServer(t, `{"ai_runtime_task_output":{"mlflow_experiment_id":"E1","mlflow_run_id":"R1"}}`, &hit)

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
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/2.2/jobs/runs/get-output" {
				runID := r.URL.Query().Get("run_id")
				_, _ = w.Write([]byte(`{"ai_runtime_task_output":{"mlflow_experiment_id":"E1","mlflow_run_id":"R` + runID + `"}}`))
				return
			}
			_, _ = w.Write([]byte(`{}`))
		}))
		t.Cleanup(srv.Close)

		retried := &jobs.Run{Tasks: []jobs.RunTask{{RunId: 99}, {RunId: 100}}}
		got := mlflowIDs(ctx, newTestWorkspaceClient(t, srv.URL), retried)
		assert.Equal(t, &mlflowIdentifiers{ExperimentID: "E1", RunID: "R100"}, got)
	})

	t.Run("a finished run falls back to an earlier attempt when the latest has none", func(t *testing.T) {
		// The last retry failed before it started, so only attempt 0 logged to MLflow.
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/2.2/jobs/runs/get-output" && r.URL.Query().Get("run_id") == "99" {
				_, _ = w.Write([]byte(`{"ai_runtime_task_output":{"mlflow_experiment_id":"E1","mlflow_run_id":"R1"}}`))
				return
			}
			_, _ = w.Write([]byte(`{}`))
		}))
		t.Cleanup(srv.Close)

		w := newTestWorkspaceClient(t, srv.URL)
		tasks := []jobs.RunTask{{RunId: 99}, {RunId: 100, AttemptNumber: 1}}
		finished := &jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated}, Tasks: tasks}
		retry0 := 0
		assert.Equal(t, &mlflowIdentifiers{ExperimentID: "E1", RunID: "R1", OlderRetry: &retry0}, mlflowIDs(ctx, w, finished))

		// A run still starting its next retry must not link to the previous attempt.
		running := &jobs.Run{State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateRunning}, Tasks: tasks}
		assert.Nil(t, mlflowIDs(ctx, w, running))

		// In a multi-task job, an earlier task is not a retry of the last one.
		multiTask := &jobs.Run{
			State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated},
			Tasks: []jobs.RunTask{{TaskKey: "train", RunId: 99}, {TaskKey: "report", RunId: 100}},
		}
		assert.Nil(t, mlflowIDs(ctx, w, multiTask))
	})

	t.Run("a finished run links no older retry when a newer one's output can't be read", func(t *testing.T) {
		// Given retry 0 logged to MLflow, but get-output fails for failingRunID.
		finished := &jobs.Run{
			State: &jobs.RunState{LifeCycleState: jobs.RunLifeCycleStateTerminated},
			Tasks: []jobs.RunTask{{RunId: 98}, {RunId: 99, AttemptNumber: 1}, {RunId: 100, AttemptNumber: 2}},
		}
		for _, failingRunID := range []string{"100", "99"} {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.URL.Path != "/api/2.2/jobs/runs/get-output":
					_, _ = w.Write([]byte(`{}`))
				case r.URL.Query().Get("run_id") == failingRunID:
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"error_code":"PERMISSION_DENIED","message":"denied"}`))
				case r.URL.Query().Get("run_id") == "98":
					_, _ = w.Write([]byte(`{"ai_runtime_task_output":{"mlflow_experiment_id":"E1","mlflow_run_id":"R0"}}`))
				default:
					_, _ = w.Write([]byte(`{}`))
				}
			}))
			t.Cleanup(srv.Close)

			// When the views pick the run's MLflow run.
			got := mlflowIDs(ctx, newTestWorkspaceClient(t, srv.URL), finished)

			// Then there is no link: the unread retry may have the newest MLflow run.
			assert.Nil(t, got, "get-output fails for task run %s", failingRunID)
		}
	})
}

func TestAiRuntimeTaskOutput(t *testing.T) {
	var hit bool
	srv := runOutputServer(t, `{"ai_runtime_task_output":{"status_message":"STATUS: Waiting for GPU capacity"}}`, &hit)
	run := &jobs.Run{Tasks: []jobs.RunTask{{RunId: 99}}}

	got := aiRuntimeTaskOutput(t.Context(), newTestWorkspaceClient(t, srv.URL), run)
	require.NotNil(t, got)
	assert.True(t, hit)
	assert.Equal(t, "STATUS: Waiting for GPU capacity", got.StatusMessage)
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

	t.Run("nil when no task run id", func(t *testing.T) {
		// Returns before any HTTP call, so the host is never contacted.
		assert.Nil(t, mlflowIDsForTask(ctx, newTestWorkspaceClient(t, "https://unused.invalid"), 0))
	})
}

func TestMLflowURLs(t *testing.T) {
	ids := &mlflowIdentifiers{ExperimentID: "E1", RunID: "R1"}
	// A trailing slash on the host must not produce a double slash in the link.
	assert.Equal(t, "https://h.test/ml/experiments/E1/runs/R1/logs", mlflowLogsURL("https://h.test/", ids))
	assert.Equal(t, "https://h.test/ml/experiments/E1/runs/R1", mlflowRunURL("https://h.test", ids))
}

func TestMLflowRunLabel(t *testing.T) {
	const runID = "0123456789abcdef0123456789abcdef"
	tests := []struct {
		name    string
		runName string
		runID   string
		width   int
		want    string
	}{
		{name: "long name keeps its end", runName: "qwen-finetune-2026-10-23-lr-3e-5", runID: runID, width: 22, want: "…ne-2026-10-23-lr-3e-5"},
		{name: "short name is shown whole", runName: "sunny-cat-42", runID: runID, width: 22, want: "sunny-cat-42"},
		{name: "name at the width is shown whole", runName: "exactly-twenty-two-chr", runID: runID, width: 22, want: "exactly-twenty-two-chr"},
		{name: "unnamed run keeps the end of its ID", runID: runID, width: 22, want: "…bcdef0123456789abcdef"},
		{name: "short ID is shown whole", runID: "short", width: 22, want: "short"},
		{name: "wide characters count as two cells", runName: "训练运行-实验-第一百二十三次", runID: runID, width: 8, want: "…十三次"},
		{name: "width of one leaves only the ellipsis", runName: "sunny-cat-42", runID: runID, width: 1, want: "…"},
		{name: "no width limit shows the whole name", runName: "qwen-finetune-2026-10-23-lr-3e-5", runID: runID, want: "qwen-finetune-2026-10-23-lr-3e-5"},
		{name: "no width limit shows the whole ID", runID: runID, want: runID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mlflowRunLabel(tt.runName, tt.runID, tt.width)
			assert.Equal(t, tt.want, got)
			if tt.width > 0 {
				assert.LessOrEqual(t, lipgloss.Width(got), tt.width)
			}
		})
	}
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
