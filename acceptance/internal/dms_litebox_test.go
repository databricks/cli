package internal

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiteboxWorkspaceRouting(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	// Stand in for the gateway transport; the subject here is the routing adapter.
	routeLitebox(server, func(r testserver.Request) any {
		return testserver.Response{StatusCode: http.StatusOK, Headers: http.Header{"X-Gateway": {"true"}}, Body: map[string]any{}}
	})

	for _, tc := range []struct {
		name, method, path, body string
		status                   int
		gateway                  bool
	}{
		{"missing file", "GET", "/api/2.0/workspace/get-status?path=/test/state.json", "", 404, false},
		{"missing file deletion", "POST", "/api/2.0/workspace/delete", `{"path":"/test/state.json"}`, 200, false},
		{"mkdir mirrors locally", "POST", "/api/2.0/workspace/mkdirs", `{"path":"/test"}`, 200, true},
		{"directory keeps remote ID", "GET", "/api/2.0/workspace/get-status?path=/test", "", 200, true},
		{"import stays local", "POST", "/api/2.0/workspace/import", `{"path":"/test/state.json","format":"RAW","content":"e30="}`, 200, false},
		{"existing file stays local", "GET", "/api/2.0/workspace/get-status?path=/test/state.json", "", 200, false},
		{"deployment node stays remote", "GET", "/api/2.0/workspace/get-status?path=/test/resources.deployment.json", "", 200, true},
		{"deployment API stays remote", "GET", "/api/2.0/bundle/deployments/123", "", 200, true},
		{"deployment deletion stays remote", "POST", "/api/2.0/workspace/delete", `{"path":"/test/resources.deployment.json"}`, 200, true},
		{"directory deletion mirrors locally", "POST", "/api/2.0/workspace/delete", `{"path":"/test","recursive":true}`, 200, true},
		{"deleted directory is absent", "GET", "/api/2.0/workspace/get-status?path=/test", "", 404, false},
		{"deleted child is absent", "GET", "/api/2.0/workspace/get-status?path=/test/state.json", "", 404, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(t.Context(), tc.method, server.URL+tc.path, strings.NewReader(tc.body))
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer test-token")
			resp, err := server.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, tc.status, resp.StatusCode, "%s", body)
			assert.Equal(t, tc.gateway, resp.Header.Get("X-Gateway") == "true")
		})
	}
}
