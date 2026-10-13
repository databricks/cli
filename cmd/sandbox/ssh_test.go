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

func TestBuildSSHArgsBaseFlags(t *testing.T) {
	got := buildSSHArgs("happy-panda-1234", "gw.example.test", "2222", "/keys/id", nil)
	// Sanity: the destination is the last arg when no extras.
	assert.Equal(t, "happy-panda-1234@gw.example.test", got[len(got)-1])
	// The base flag set is fixed; spot-check a couple.
	assert.Contains(t, got, "-i")
	assert.Contains(t, got, "/keys/id")
	assert.Contains(t, got, "-p")
	assert.Contains(t, got, "2222")
}

func TestBuildSSHArgsQuoting(t *testing.T) {
	for _, tc := range []struct {
		name      string
		extraArgs []string
		// expected is what we expect at the end of args, after the destination.
		expected []string
	}{
		{
			name:      "no extras",
			extraArgs: nil,
			expected:  nil,
		},
		{
			name:      "single safe word — passed through",
			extraArgs: []string{"uname"},
			expected:  []string{"uname"},
		},
		{
			name:      "single string with spaces — passed through so the remote shell can split it",
			extraArgs: []string{"echo hello-from-string"},
			expected:  []string{"echo hello-from-string"},
		},
		{
			name:      "single string with a pipe — passed through to the remote shell",
			extraArgs: []string{"cat /etc/os-release | head -3"},
			expected:  []string{"cat /etc/os-release | head -3"},
		},
		{
			name:      "multi safe words — no quoting",
			extraArgs: []string{"ls", "-la", "/tmp"},
			expected:  []string{"ls", "-la", "/tmp"},
		},
		{
			name: "bash -c '<cmd>' — third arg gets quoted",
			// The whole point: without the quoting, the remote shell
			// re-splits "bash -c echo hi" and bash's -c eats just "echo".
			extraArgs: []string{"bash", "-c", "echo hi"},
			expected:  []string{"bash", "-c", "'echo hi'"},
		},
		{
			name:      "arg with single quote — escaped, not lost",
			extraArgs: []string{"echo", "it's fine"},
			expected:  []string{"echo", `'it'\''s fine'`},
		},
		{
			name:      "glob pattern — quoted so the remote (not the local) shell expands it",
			extraArgs: []string{"find", ".", "-name", "*.txt"},
			expected:  []string{"find", ".", "-name", "'*.txt'"},
		},
		{
			name:      "ssh -o key=val style — gets quoted because of '='; harmless given current arg ordering",
			extraArgs: []string{"-o", "StrictHostKeyChecking=no"},
			expected:  []string{"-o", "'StrictHostKeyChecking=no'"},
		},
		{
			name:      "empty arg gets explicit empty quotes (not lost)",
			extraArgs: []string{"sh", "-c", ""},
			expected:  []string{"sh", "-c", "''"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := buildSSHArgs("id", "host", "2222", "/k", tc.extraArgs)
			// Trim everything up to and including the destination so we
			// can assert just on the appended extras.
			dest := "id@host"
			cut := -1
			for i, a := range got {
				if a == dest {
					cut = i + 1
					break
				}
			}
			assert.GreaterOrEqual(t, cut, 0, "destination not found in args")
			tail := got[cut:]
			if tc.expected == nil {
				assert.Empty(t, tail)
			} else {
				assert.Equal(t, tc.expected, tail)
			}
		})
	}
}
