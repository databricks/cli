package aircmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runGetServer serves one runs/get body and a stub for the SDK config probe.
func runGetServer(t *testing.T, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.2/jobs/runs/get" {
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestFetchJobRunParsesAiRuntimeTask(t *testing.T) {
	body := `{
		"run_id": 5,
		"tasks": [{
			"ai_runtime_task": {
				"experiment": "my-exp",
				"deployments": [{
					"command_path": "/Workspace/Users/me/.air/cli_launch/my-exp/my-exp_abc/command.sh",
					"compute": {"accelerator_count": 1, "accelerator_type": "GPU_1xA10"}
				}]
			}
		}]
	}`
	srv := runGetServer(t, body)

	run, err := fetchJobRun(t.Context(), newTestWorkspaceClient(t, srv.URL), 5)
	require.NoError(t, err)
	assert.Equal(t, "my-exp", jobExperiment(run))
	gpuType, count := jobCompute(run)
	assert.Equal(t, "GPU_1xA10", gpuType)
	assert.Equal(t, 1, count)
}
