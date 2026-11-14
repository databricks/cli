package aircmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
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

type runGetResponse struct {
	status int
	body   string
}

// runGetByIDServer serves runs/get per run_id and a stub for the SDK config
// probe; ids missing from responses return a run with that id.
func runGetByIDServer(t *testing.T, responses map[string]runGetResponse) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/2.2/jobs/runs/get" {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		id := r.URL.Query().Get("run_id")
		resp, ok := responses[id]
		if !ok {
			resp = runGetResponse{status: http.StatusOK, body: `{"run_id": ` + id + `}`}
		}
		w.WriteHeader(resp.status)
		_, _ = w.Write([]byte(resp.body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runIDs(runs []*jobs.Run) []int64 {
	ids := make([]int64, len(runs))
	for i, r := range runs {
		ids[i] = r.RunId
	}
	return ids
}

func TestHydrateJobRunsDropsUnviewableAndPurgedRuns(t *testing.T) {
	srv := runGetByIDServer(t, map[string]runGetResponse{
		"2": {http.StatusBadRequest, `{"error_code": "INVALID_PARAMETER_VALUE", "message": "Run 2 does not exist."}`},
		"4": {http.StatusForbidden, `{"error_code": "PERMISSION_DENIED", "message": "denied"}`},
		"5": {http.StatusNotFound, `{"error_code": "NOT_FOUND", "message": "not found"}`},
	})

	runs, err := hydrateJobRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), []int64{1, 2, 3, 4, 5, 6})
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 3, 6}, runIDs(runs))
}

func TestHydrateJobRunsFailsOnOtherErrors(t *testing.T) {
	tests := []struct {
		name    string
		resp    runGetResponse
		wantErr string
	}{
		{
			name:    "internal error",
			resp:    runGetResponse{http.StatusInternalServerError, `{"error_code": "INTERNAL_ERROR", "message": "boom"}`},
			wantErr: "failed to get run 2: boom",
		},
		{
			name:    "other invalid parameter",
			resp:    runGetResponse{http.StatusBadRequest, `{"error_code": "INVALID_PARAMETER_VALUE", "message": "run_id must be positive"}`},
			wantErr: "failed to get run 2: run_id must be positive",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := runGetByIDServer(t, map[string]runGetResponse{"2": tt.resp})

			_, err := hydrateJobRuns(t.Context(), newTestWorkspaceClient(t, srv.URL), []int64{1, 2, 3})
			assert.EqualError(t, err, tt.wantErr)
		})
	}
}
