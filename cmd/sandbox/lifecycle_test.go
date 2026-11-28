package sandbox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSandboxRejectsAccountOnlySentinel(t *testing.T) {
	for _, accountID := range []string{"", "test-account"} {
		t.Run(accountID, func(t *testing.T) {
			server := testserver.New(t)
			testserver.AddDefaultHandlers(server)
			w, err := databricks.NewWorkspaceClient(&databricks.Config{
				Host: server.URL, Token: "testtoken", AccountID: accountID, WorkspaceID: profile.WorkspaceIDNone,
			})
			require.NoError(t, err)
			server.RequestCallback = func(r *testserver.Request) { t.Errorf("unexpected request %s", r.URL.Path) }
			_, err = newSandboxAPI(w)
			require.ErrorContains(t, err, "require a workspace")
			assert.Equal(t, accountID, w.Config.AccountID)
			assert.Equal(t, profile.WorkspaceIDNone, w.Config.WorkspaceID)
			assert.Equal(t, server.URL, w.Config.Host)
			assert.Equal(t, "testtoken", w.Config.Token)
		})
	}
}

func TestSandboxRejectsUnsafeIDsBeforeRequests(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	api, err := newSandboxAPI(w)
	require.NoError(t, err)
	server.RequestCallback = func(r *testserver.Request) { t.Errorf("unexpected request %s", r.URL.Path) }
	for _, id := range []string{"", ".", "..", "a/b", `a\b`, "a?b", "a#b", "a%2fb", "a b"} {
		_, err = api.get(t.Context(), id)
		require.ErrorContains(t, err, "invalid sandbox ID")
		_, err = api.update(t.Context(), id, nil, nil, nil)
		require.ErrorContains(t, err, "invalid sandbox ID")
		_, err = api.start(t.Context(), id)
		require.ErrorContains(t, err, "invalid sandbox ID")
		_, err = api.stop(t.Context(), id)
		require.ErrorContains(t, err, "invalid sandbox ID")
		require.ErrorContains(t, api.delete(t.Context(), id), "invalid sandbox ID")
	}
}

// Custom replies exercise creation uncertainty and provisioning cancellation,
// which require control over in-flight requests rather than SDK serialization.
func newCreateFaultAPI(t *testing.T, workspaceID string, handler http.HandlerFunc) *sandboxAPI {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			_, _ = io.WriteString(w, `{}`)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token", WorkspaceID: workspaceID})
	require.NoError(t, err)
	api, err := newSandboxAPI(w)
	require.NoError(t, err)
	assert.Equal(t, workspaceID, w.Config.WorkspaceID, "API construction must preserve the workspace client config")
	return api
}

func TestPublicSandboxCreateReadiness(t *testing.T) {
	for _, tc := range []struct{ name, state, wantErr string }{
		{name: "pending reaches running", state: "RUNNING"},
		{name: "provisioning failure", state: "STOPPED", wantErr: `unexpected state "Stopped" while provisioning`},
		{name: "cancel pending create", state: "PENDING", wantErr: "context deadline exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gets := 0
			api := newCreateFaultAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
				state := "PENDING"
				if r.Method == http.MethodGet {
					gets++
					state = tc.state
					assert.Equal(t, "/api/2.0/sandboxes/test-id", r.URL.Path)
				}
				_, _ = io.WriteString(w, `{"name":"sandboxes/test-id","status":{"state":"SANDBOX_STATE_`+state+`"}}`)
			})
			ctx, cancel := context.WithTimeout(env.WithUserHomeDir(t.Context(), t.TempDir()), time.Second)
			defer cancel()
			got, err := createAndWait(ctx, api, "", nil)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				assert.ErrorContains(t, err, "sandbox test-id was created but is not ready")
				assert.Nil(t, got)
			} else {
				require.NoError(t, err)
				assert.Equal(t, "Running", sandboxStatus(got))
			}
			assert.Equal(t, 1, gets)
			assert.Equal(t, []cachedSandbox{{ID: "test-id"}}, getSandboxes(ctx, api.c.Config.Host))
		})
	}
}

func TestPublicSandboxCreateFailureIncludesGeneratedID(t *testing.T) {
	var id string
	calls := 0
	api := newCreateFaultAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
		calls++
		id = r.URL.Query().Get("sandbox_id")
		w.WriteHeader(http.StatusConflict)
		_, _ = io.WriteString(w, `{"error_code":"RESOURCE_ALREADY_EXISTS","message":"already exists"}`)
	})
	_, err := createAndWait(t.Context(), api, "project", nil)
	require.Error(t, err)
	assert.Regexp(t, `^[a-z]+-[a-z]+-[1-9][0-9]{3}$`, id)
	assert.ErrorContains(t, err, id)
	var apiError *apierr.APIError
	require.ErrorAs(t, err, &apiError)
	assert.Equal(t, http.StatusConflict, apiError.StatusCode)
	assert.Equal(t, 1, calls, "a collision must remain an actionable error without creating another sandbox")
}

func TestPublicSandboxCancelledCreateIncludesRequestedID(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	release := make(chan struct{})
	defer close(release)
	var id string
	api := newCreateFaultAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
		id = r.URL.Query().Get("sandbox_id")
		cancel()
		<-release
	})
	_, err := api.create(ctx, "project")
	require.Error(t, err)
	assert.Regexp(t, `^[a-z]+-[a-z]+-[1-9][0-9]{3}$`, id)
	assert.ErrorContains(t, err, id)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestCreatePreservesIDLabelAndPersistencePolicy(t *testing.T) {
	for _, label := range []string{"", "my-project", "My Project/Label"} {
		t.Run(label, func(t *testing.T) {
			server := testserver.New(t)
			testserver.AddDefaultHandlers(server)
			var requestedID string
			var request map[string]any
			server.RequestCallback = func(r *testserver.Request) {
				if r.URL.Path == "/api/2.0/sandboxes" || r.URL.Path == "/api/2.0/sandboxes/"+requestedID {
					assert.Contains(t, r.Headers.Get("User-Agent"), "cli/"+build.GetInfo().Version)
				}
				if r.Method == http.MethodPost && r.URL.Path == "/api/2.0/sandboxes" {
					requestedID = r.URL.Query().Get("sandbox_id")
					require.NoError(t, json.Unmarshal(r.Body, &request))
				}
			}
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
			require.NoError(t, err)
			api, err := newSandboxAPI(w)
			require.NoError(t, err)
			sb, err := api.create(t.Context(), label)
			require.NoError(t, err)
			assert.Regexp(t, `^[a-z]+-[a-z]+-[1-9][0-9]{3}$`, requestedID)
			assert.Equal(t, "sandboxes/"+requestedID, sb.Name)
			assert.Equal(t, label, sb.DisplayName)
			expected := map[string]any{"spec": map[string]any{"compute": map[string]any{"inactivity_timeout": "600s"}}}
			if label != "" {
				expected["display_name"] = label
			}
			assert.Equal(t, expected, request, "the CLI must leave persistence omitted")
			_, err = api.get(t.Context(), requestedID)
			require.NoError(t, err)
			_, err = api.list(t.Context())
			require.NoError(t, err)
		})
	}
}
