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
	// A cursor that always returns the same token must not loop forever.
	page := `{"training_workflows":[{"job_run_id":1,"submit_time":"2023-11-14T22:13:20Z"}],"next_page_token":"tok"}`
	var hit bool
	srv := indexServer(t, &hit, page)

	refs, err := listAiTrainingWorkflows(t.Context(), newTestWorkspaceClient(t, srv.URL), false)
	require.NoError(t, err)
	require.Len(t, refs, 2) // first page + one more before the repeated token stops it
}
