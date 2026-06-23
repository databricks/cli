package aircmd

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListTrainingWorkflows(t *testing.T) {
	var gotQuery url.Values
	body := `{
		"training_workflows": [
			{
				"job_run_id": "123",
				"metadata": {"creator_name": "me@example.com"},
				"spec": {"compute": {"hardware_accelerator_type": "GPU_8xH100", "accelerator_count": 8}},
				"status": {
					"state": "TRAINING_WORKFLOW_STATE_RUNNING",
					"start_time": "2026-06-05T17:32:39.791Z",
					"job": {"name": "my-run"},
					"mlflow": {"experiment": "/Users/me@example.com/exp", "experiment_id": "E1", "run_id": "R1"}
				}
			}
		],
		"next_page_token": "tok"
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == listWorkflowsPath {
			gotQuery = r.URL.Query()
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	resp, err := listTrainingWorkflows(t.Context(), newTestWorkspaceClient(t, srv.URL), map[string]any{
		"creator_name": "me@example.com",
		"active_only":  true,
	})
	require.NoError(t, err)
	require.Len(t, resp.TrainingWorkflows, 1)
	assert.Equal(t, "tok", resp.NextPageToken)

	wf := resp.TrainingWorkflows[0]
	assert.Equal(t, "123", wf.JobRunID)
	assert.Equal(t, "me@example.com", wf.Metadata.CreatorName)
	assert.Equal(t, "GPU_8xH100", wf.Spec.Compute.HardwareAcceleratorType)
	assert.Equal(t, 8, wf.Spec.Compute.AcceleratorCount)
	assert.Equal(t, "TRAINING_WORKFLOW_STATE_RUNNING", wf.Status.State)
	assert.Equal(t, "my-run", wf.Status.Job.Name)
	assert.Equal(t, "E1", wf.Status.Mlflow.ExperimentID)
	assert.Equal(t, "R1", wf.Status.Mlflow.RunID)

	// The query map is serialized into the GET query string.
	assert.Equal(t, "me@example.com", gotQuery.Get("creator_name"))
	assert.Equal(t, "true", gotQuery.Get("active_only"))
}
