package aircmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLogsSnapshotFailedRetryWithoutLogs(t *testing.T) {
	// Only attempt 0 ran: retries 1 and 2 failed before they started, with no
	// logs or MLflow output of their own. The pane still links the MLflow run
	// that attempt 0 logged to.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/logs"):
			_, _ = w.Write([]byte(`{"log_records": []}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(`{"run_id": 123, "state": {"life_cycle_state": "TERMINATED", "result_state": "FAILED", "state_message": "Out of capacity"},
				"tasks": [{"run_id": 456, "attempt_number": 0}, {"run_id": 457, "attempt_number": 1}, {"run_id": 458, "attempt_number": 2}]}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get-output" && r.URL.Query().Get("run_id") == "456":
			_, _ = w.Write([]byte(`{"ai_runtime_task_output": {"mlflow_experiment_id": "E1", "mlflow_run_id": "R1"}}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)

	got, err := runLogsSnapshot(t.Context(), newTestWorkspaceClient(t, srv.URL), 123)
	require.NoError(t, err)
	assert.Equal(t, "No logs available for run 123. Run terminated in state FAILED: Out of capacity. "+
		"Earlier retries may have logs; pass --retry N (0 through 1) to see them\n"+
		"\nMLflow URL (retry 0): "+srv.URL+"/ml/experiments/E1/runs/R1/logs\n", got)
}

func TestListRowAndPanesLinkTheSameOlderRetry(t *testing.T) {
	// Given a failed run whose retries 0 and 1 logged to MLflow and whose last
	// retry failed before it started.
	t.Setenv("DATABRICKS_CACHE_DIR", t.TempDir())
	run := indexRun(7, 1000_000)
	run.State.ResultState = jobs.RunResultStateFailed
	task := run.Tasks[0]
	run.Tasks = nil
	for attempt, taskRunID := range []int64{70, 71, 72} {
		task.RunId, task.AttemptNumber = taskRunID, attempt
		run.Tasks = append(run.Tasks, task)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case aiTrainingWorkflowsPath:
			_, _ = w.Write([]byte(`{"training_workflows": [{"job_run_id": "7", "submit_time": {"seconds": 1000}}]}`))
		case "/api/2.2/jobs/runs/get":
			_ = json.NewEncoder(w).Encode(run)
		case "/api/2.2/jobs/runs/get-output":
			switch r.URL.Query().Get("run_id") {
			case "70":
				_, _ = w.Write([]byte(`{"ai_runtime_task_output": {"mlflow_experiment_id": "E1", "mlflow_run_id": "R0"}}`))
			case "71":
				_, _ = w.Write([]byte(`{"ai_runtime_task_output": {"mlflow_experiment_id": "E1", "mlflow_run_id": "R1"}}`))
			default:
				_, _ = w.Write([]byte(`{}`))
			}
		case "/api/2.0/mlflow/runs/get":
			_, _ = w.Write([]byte(`{"run": {"info": {"run_name": "train-` + r.URL.Query().Get("run_id") + `"}}}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	ctx := cmdio.MockDiscard(t.Context())
	w := newTestWorkspaceClient(t, srv.URL)
	wantURL := srv.URL + "/ml/experiments/E1/runs/R1/logs"

	// When the list, the info pane and the logs pane render the run.
	rows, err := newRunFetcher(ctx, w, listQuery{
		userFilter: "me@example.com", currentUser: "me@example.com", limit: 10, fetchMLflow: true,
	}).next(10)
	require.NoError(t, err)
	info, err := runDetailText(ctx, w, 7)
	require.NoError(t, err)
	logs, err := runLogsSnapshot(ctx, w, 7)
	require.NoError(t, err)

	// Then each links retry 1's MLflow run and names that retry.
	require.Len(t, rows, 1)
	assert.Equal(t, wantURL, rows[0].MLflowURL)
	assert.Equal(t, "train-R1 (retry 1)", rows[0].MLflowLabel)
	assert.Contains(t, info, "train-R1 (retry 1)")
	assert.Contains(t, info, "MLflow URL: "+srv.URL+"/ml/experiments/E1/runs/R1\n")
	assert.Contains(t, logs, "\nMLflow URL (retry 1): "+wantURL+"\n")
}
