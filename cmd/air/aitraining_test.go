package aircmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSubmitTimeMs(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want int64
	}{
		{"rfc3339", `"2023-11-14T22:13:20Z"`, 1700000000000},
		{"rfc3339 offset", `"2023-11-14T22:13:20+00:00"`, 1700000000000},
		{"seconds and nanos", `{"seconds": 1700000000, "nanos": 500000000}`, 1700000000500},
		{"seconds only", `{"seconds": 1700000000}`, 1700000000000},
		{"empty", ``, 0},
		{"garbage string", `"not-a-time"`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, parseSubmitTimeMs(json.RawMessage(tc.raw)))
		})
	}
}

func TestListWorkspaceComputeOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			_, _ = w.Write([]byte(`{}`))
			return
		}
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, computeOptionsPath, r.URL.Path)
		assert.Equal(t, "123", r.Header.Get("X-Databricks-Workspace-Id"))
		_, _ = w.Write([]byte(`{"compute_options":[{"hardware_accelerator":"GPU_8xH100","display_name":"8x H100","multi_node_supported":true,"per_node_accelerator_count":8,"launch_stage":"PUBLIC_PREVIEW"}]}`))
	}))
	t.Cleanup(srv.Close)
	w := newTestWorkspaceClient(t, srv.URL)
	w.Config.WorkspaceID = "123"

	options, err := listWorkspaceComputeOptions(t.Context(), w)
	require.NoError(t, err)
	require.Len(t, options, 1)
	assert.Equal(t, "GPU_8xH100", options[0].HardwareAccelerator)
	assert.Equal(t, "8x H100", options[0].DisplayName)
	assert.True(t, *options[0].MultiNodeSupported)
	assert.Equal(t, 8, *options[0].PerNodeAcceleratorCount)
	assert.Equal(t, "PUBLIC_PREVIEW", options[0].LaunchStage)
}

// indexServer serves paginated AiTrainingService responses, one body per call,
// tracking whether the index was hit.
func indexServer(t *testing.T, hit *bool, bodies ...string) *httptest.Server {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == aiTrainingWorkflowsPath {
			*hit = true
			body := bodies[min(call, len(bodies)-1)]
			call++
			_, _ = w.Write([]byte(body))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestListAiTrainingWorkflowsPaginates(t *testing.T) {
	page1 := `{"training_workflows":[{"job_run_id":"1","submit_time":"2023-11-14T22:13:20Z"}],"next_page_token":"tok"}`
	page2 := `{"training_workflows":[{"job_run_id":2,"submit_time":{"seconds":1700000100}}]}`
	var hit bool
	srv := indexServer(t, &hit, page1, page2)

	refs, err := listAiTrainingWorkflows(t.Context(), newTestWorkspaceClient(t, srv.URL), false)
	require.NoError(t, err)
	require.Len(t, refs, 2)
	assert.Equal(t, int64(1), refs[0].jobRunID)
	assert.Equal(t, int64(1700000000000), refs[0].submitTimeMs)
	assert.Equal(t, int64(2), refs[1].jobRunID)
}

func TestListAiTrainingWorkflowsStopsOnRepeatedToken(t *testing.T) {
	// A cursor that always returns the same token must not loop forever. The
	// repeated id is also deduped, so only one ref survives.
	page := `{"training_workflows":[{"job_run_id":1,"submit_time":"2023-11-14T22:13:20Z"}],"next_page_token":"tok"}`
	var hit bool
	srv := indexServer(t, &hit, page)

	refs, err := listAiTrainingWorkflows(t.Context(), newTestWorkspaceClient(t, srv.URL), false)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, int64(1), refs[0].jobRunID)
}

func TestListAiTrainingWorkflowsDedupesIDs(t *testing.T) {
	// The same job_run_id on multiple pages must be counted once, so the
	// newest-limit truncation doesn't silently return fewer unique runs.
	page1 := `{"training_workflows":[{"job_run_id":1,"submit_time":"2023-11-14T22:13:20Z"},{"job_run_id":2,"submit_time":"2023-11-14T22:13:21Z"}],"next_page_token":"tok"}`
	page2 := `{"training_workflows":[{"job_run_id":2,"submit_time":"2023-11-14T22:13:21Z"},{"job_run_id":3,"submit_time":"2023-11-14T22:13:22Z"}]}`
	var hit bool
	srv := indexServer(t, &hit, page1, page2)

	refs, err := listAiTrainingWorkflows(t.Context(), newTestWorkspaceClient(t, srv.URL), false)
	require.NoError(t, err)
	require.Len(t, refs, 3)
	got := []int64{refs[0].jobRunID, refs[1].jobRunID, refs[2].jobRunID}
	assert.ElementsMatch(t, []int64{1, 2, 3}, got)
}
