package sandbox

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureRunningLifecycle(t *testing.T) {
	type requestStep struct {
		method string
		path   string
		status string
	}

	for _, tc := range []struct {
		name          string
		initialStatus string
		steps         []requestStep
		wantStatus    string
		wantErr       string
		wantStarts    int32
	}{
		{
			name:          "stopping waits for stopped before starting",
			initialStatus: "Stopping",
			steps: []requestStep{
				{method: http.MethodGet, path: sandboxPath("test-id"), status: "Stopped"},
				{method: http.MethodPost, path: sandboxPath("test-id") + "/start", status: "Pending"},
				{method: http.MethodGet, path: sandboxPath("test-id"), status: "Running"},
			},
			wantStatus: "Running",
			wantStarts: 1,
		},
		{
			name:          "stopped starts normally",
			initialStatus: "Stopped",
			steps: []requestStep{
				{method: http.MethodPost, path: sandboxPath("test-id") + "/start", status: "Pending"},
				{method: http.MethodGet, path: sandboxPath("test-id"), status: "Running"},
			},
			wantStatus: "Running",
			wantStarts: 1,
		},
		{
			name:          "running is a no-op",
			initialStatus: "Running",
			wantStatus:    "Running",
		},
		{
			name:          "stopped after start remains an error",
			initialStatus: "Stopped",
			steps: []requestStep{
				{method: http.MethodPost, path: sandboxPath("test-id") + "/start", status: "Pending"},
				{method: http.MethodGet, path: sandboxPath("test-id"), status: "Stopped"},
			},
			wantErr:    `sandbox test-id reached unexpected state "Stopped" while starting`,
			wantStarts: 1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requestCount atomic.Int32
			var startCount atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/.well-known/databricks-config" {
					assert.NoError(t, json.NewEncoder(w).Encode(map[string]string{
						"oidc_endpoint": "https://workspace.example.test/oidc",
					}))
					return
				}

				index := int(requestCount.Add(1) - 1)
				if index >= len(tc.steps) {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
					http.Error(w, "unexpected request", http.StatusInternalServerError)
					return
				}

				step := tc.steps[index]
				assert.Equal(t, step.method, r.Method)
				assert.Equal(t, step.path, r.URL.Path)
				if r.Method == http.MethodPost && r.URL.Path == sandboxPath("test-id")+"/start" {
					startCount.Add(1)
				}
				assert.NoError(t, json.NewEncoder(w).Encode(sandboxEntry{
					SandboxID: "test-id",
					Status:    step.status,
				}))
			}))
			t.Cleanup(server.Close)

			workspaceClient, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host:  server.URL,
				Token: "test-token",
			})
			require.NoError(t, err)
			api, err := newSandboxAPI(workspaceClient)
			require.NoError(t, err)

			ctx := cmdio.MockDiscard(t.Context())
			got, err := ensureRunning(ctx, api, "test-id", tc.initialStatus)
			if tc.wantErr != "" {
				require.EqualError(t, err, tc.wantErr)
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				assert.Equal(t, tc.wantStatus, got.Status)
			}
			assert.Equal(t, int32(len(tc.steps)), requestCount.Load())
			assert.Equal(t, tc.wantStarts, startCount.Load())
		})
	}
}
