package aircmd

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunLogsSnapshotFailedRetryWithoutLogs(t *testing.T) {
	// The last retry failed before it started: no logs and no MLflow output of
	// its own. The pane explains that and still links the MLflow run that
	// attempt 0 logged to.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/logs"):
			_, _ = w.Write([]byte(`{"log_records": []}`))
		case r.URL.Path == "/api/2.2/jobs/runs/get":
			_, _ = w.Write([]byte(`{"run_id": 123, "state": {"life_cycle_state": "TERMINATED", "result_state": "FAILED", "state_message": "Out of capacity"},
				"tasks": [{"run_id": 456, "attempt_number": 0}, {"run_id": 457, "attempt_number": 1}]}`))
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
		"Earlier retries may have logs; pass --retry 0 to see them\n"+
		"\nMLflow URL: "+srv.URL+"/ml/experiments/E1/runs/R1/artifacts/logs/node_0\n", got)
}
