package sandbox

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Exercise the actual SDK serialization and transport; only the remote service
// is replaced, so these tests catch changes to the public wire contract.
func newPublicAPI(t *testing.T, workspaceID string, handler http.HandlerFunc) *sandboxAPI {
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

const publicRunning = `{"name":"sandboxes/test-id","display_name":"project","status":{"state":"SANDBOX_STATE_RUNNING"},"spec":{"compute":{"inactivity_timeout":"900s"}}}`

func TestPublicSandboxCreateAndJSON(t *testing.T) {
	for _, name := range []string{"", "project", "My Project/Label"} {
		t.Run(name, func(t *testing.T) {
			var requestedID string
			api := newPublicAPI(t, "123", func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "/api/2.0/sandboxes", r.URL.Path)
				assert.Equal(t, "123", r.Header.Get(auth.WorkspaceIDHeader))
				id := r.URL.Query().Get("sandbox_id")
				requestedID = id
				assert.Regexp(t, `^[a-z]+-[a-z]+-[1-9][0-9]{3}$`, id)
				assert.LessOrEqual(t, len(id), 63)
				assert.NotEqual(t, name, id)
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				want := `{"spec":{"compute":{"inactivity_timeout":"600s"}}}`
				if name != "" {
					want = `{"display_name":"` + name + `","spec":{"compute":{"inactivity_timeout":"600s"}}}`
				}
				// Comparing the complete body also protects omitted persistence settings.
				assert.JSONEq(t, want, string(body))
				_, _ = io.WriteString(w, `{"name":"sandboxes/`+id+`","status":{"state":"SANDBOX_STATE_RUNNING"}}`)
			})
			got, err := api.create(t.Context(), name)
			require.NoError(t, err)
			data, err := json.Marshal(got)
			require.NoError(t, err)
			assert.JSONEq(t, `{"sandboxId":"`+requestedID+`","status":"Running"}`, string(data))
		})
	}
}

func TestPublicSandboxGetPreservesCLIRepresentation(t *testing.T) {
	api := newPublicAPI(t, profile.WorkspaceIDNone, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/2.0/sandboxes/test-id", r.URL.Path)
		assert.Empty(t, r.Header.Get(auth.WorkspaceIDHeader))
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.Empty(t, body)
		_, _ = io.WriteString(w, publicRunning)
	})
	got, err := api.get(t.Context(), "test-id")
	require.NoError(t, err)
	data, err := json.Marshal(got)
	require.NoError(t, err)
	assert.JSONEq(t, `{"sandboxId":"test-id","status":"Running","name":"project","idleTimeout":"900s","noAutostop":false}`, string(data))
}

func TestPublicSandboxUpdateMasksAndPresence(t *testing.T) {
	empty := ""
	label := "new label"
	zero := int64(0)
	timeout := int64(900)
	disabled := true
	enabled := false
	for _, tc := range []struct {
		name       string
		label      *string
		timeout    *int64
		disabled   *bool
		mask, body string
	}{
		{name: "rename only", label: &label, mask: "display_name", body: `{"display_name":"new label"}`},
		{name: "clear name", label: &empty, mask: "display_name", body: `{"display_name":""}`},
		{name: "timeout only", timeout: &timeout, mask: "spec.compute.inactivity_timeout", body: `{"spec":{"compute":{"inactivity_timeout":"900s"}}}`},
		{name: "clear timeout", timeout: &zero, mask: "spec.compute.inactivity_timeout", body: `{"spec":{"compute":{"inactivity_timeout":"0s"}}}`},
		{name: "disable", disabled: &disabled, mask: "spec.compute.inactivity_timeout", body: `{"spec":{"compute":{"inactivity_timeout":"0s"}}}`},
		{name: "reenable at existing CLI default", disabled: &enabled, mask: "spec.compute.inactivity_timeout", body: `{"spec":{"compute":{"inactivity_timeout":"600s"}}}`},
		{name: "disable takes precedence", timeout: &timeout, disabled: &disabled, mask: "spec.compute.inactivity_timeout", body: `{"spec":{"compute":{"inactivity_timeout":"0s"}}}`},
		{name: "explicit zero takes precedence over reenable", timeout: &zero, disabled: &enabled, mask: "spec.compute.inactivity_timeout", body: `{"spec":{"compute":{"inactivity_timeout":"0s"}}}`},
		{name: "combined", label: &label, timeout: &timeout, disabled: &enabled, mask: "display_name,spec.compute.inactivity_timeout", body: `{"display_name":"new label","spec":{"compute":{"inactivity_timeout":"900s"}}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := newPublicAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodPatch, r.Method)
				assert.Equal(t, "/api/2.0/sandboxes/test-id", r.URL.Path)
				assert.Equal(t, tc.mask, r.URL.Query().Get("update_mask"))
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.JSONEq(t, tc.body, string(body))
				_, _ = io.WriteString(w, publicRunning)
			})
			got, err := api.update(t.Context(), "test-id", tc.label, tc.timeout, tc.disabled)
			require.NoError(t, err)
			assert.Equal(t, "Running", got.Status)
		})
	}
}

func TestPublicSandboxReadCompatibility(t *testing.T) {
	for _, tc := range []struct{ name, response, want string }{
		{name: "unnamed", response: `{"name":"sandboxes/test-id"}`, want: `{"sandboxId":"test-id","status":"Unknown","name":"test-id"}`},
		{name: "blank label", response: `{"name":"sandboxes/test-id","display_name":"  "}`, want: `{"sandboxId":"test-id","status":"Unknown","name":"test-id"}`},
		{name: "trimmed label", response: `{"name":"sandboxes/test-id","display_name":" project "}`, want: `{"sandboxId":"test-id","status":"Unknown","name":"project"}`},
		{name: "unknown state", response: `{"name":"sandboxes/test-id","status":{"state":"SANDBOX_STATE_UNSPECIFIED"}}`, want: `{"sandboxId":"test-id","status":"Unknown","name":"test-id"}`},
		{name: "disabled autostop", response: `{"name":"sandboxes/test-id","spec":{"compute":{"inactivity_timeout":"0s"}}}`, want: `{"sandboxId":"test-id","status":"Unknown","name":"test-id","idleTimeout":"0s","noAutostop":true}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := newPublicAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.WriteString(w, tc.response)
			})
			got, err := api.get(t.Context(), "test-id")
			require.NoError(t, err)
			data, err := json.Marshal(got)
			require.NoError(t, err)
			assert.JSONEq(t, tc.want, string(data))
		})
	}
}

func TestPublicSandboxPagination(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty continuation page", true: "later page error"}[fail], func(t *testing.T) {
			requests := 0
			api := newPublicAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
				requests++
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/api/2.0/sandboxes", r.URL.Path)
				assert.Equal(t, "100", r.URL.Query().Get("page_size"))
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.Empty(t, body)
				switch requests {
				case 1:
					assert.Empty(t, r.URL.Query().Get("page_token"))
					_, _ = io.WriteString(w, `{"next_page_token":"second","sandboxes":[]}`)
				case 2:
					assert.Equal(t, "second", r.URL.Query().Get("page_token"))
					if fail {
						w.WriteHeader(http.StatusForbidden)
						_, _ = io.WriteString(w, `{"error_code":"PERMISSION_DENIED","message":"denied"}`)
						return
					}
					_, _ = io.WriteString(w, `{"sandboxes":[`+publicRunning+`],"next_page_token":"third"}`)
				case 3:
					assert.Equal(t, "third", r.URL.Query().Get("page_token"))
					_, _ = io.WriteString(w, `{"sandboxes":[{"name":"sandboxes/stopped","status":{"state":"SANDBOX_STATE_STOPPED"}}]}`)
				default:
					t.Errorf("unexpected page %d", requests)
				}
			})
			got, err := api.list(t.Context())
			if fail {
				require.ErrorIs(t, err, apierr.ErrPermissionDenied)
				assert.Nil(t, got)
				return
			}
			require.NoError(t, err)
			require.Len(t, got, 2)
			assert.Equal(t, "test-id", got[0].SandboxID)
			assert.Equal(t, "Stopped", got[1].Status)
			assert.Equal(t, 3, requests)
		})
	}
}

func TestPublicSandboxLifecycleRequests(t *testing.T) {
	var calls []string
	api := newPublicAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		if r.Method == http.MethodGet || r.Method == http.MethodDelete {
			assert.Empty(t, body)
		} else {
			assert.JSONEq(t, `{}`, string(body))
		}
		state := "STOPPING"
		if r.Method == http.MethodGet {
			state = "STOPPED"
		}
		if strings.HasSuffix(r.URL.Path, "/start") {
			state = "PENDING"
		}
		if r.Method != http.MethodDelete {
			_, _ = io.WriteString(w, `{"name":"sandboxes/test-id","status":{"state":"SANDBOX_STATE_`+state+`"}}`)
		}
	})
	stopped, err := api.stop(t.Context(), "test-id")
	require.NoError(t, err)
	assert.Equal(t, "Stopping", stopped.Status)
	started, err := api.start(t.Context(), "test-id")
	require.NoError(t, err)
	assert.Equal(t, "Creating", started.Status)
	require.NoError(t, api.delete(t.Context(), "test-id"))
	assert.Equal(t, []string{"POST /api/2.0/sandboxes/test-id/stop", "GET /api/2.0/sandboxes/test-id", "POST /api/2.0/sandboxes/test-id/start", "DELETE /api/2.0/sandboxes/test-id"}, calls)
}

func TestPublicSandboxStartAlreadyRunning(t *testing.T) {
	api := newPublicAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = io.WriteString(w, publicRunning)
	})
	got, err := api.start(t.Context(), "test-id")
	require.NoError(t, err)
	assert.Equal(t, "Running", got.Status)
}

func TestPublicSandboxErrors(t *testing.T) {
	for _, status := range []int{400, 403, 404, 409, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var calls atomic.Int32
			api := newPublicAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `{"error_code":"ERROR","message":"public API error"}`)
			})
			_, err := api.get(t.Context(), "test-id")
			require.Error(t, err)
			if status == 503 {
				assert.EqualError(t, err, "the Databricks Sandbox feature is not available in your region. See "+sandboxDocsURL)
				assert.Equal(t, int32(max503Attempts), calls.Load())
				return
			}
			var apiError *apierr.APIError
			require.ErrorAs(t, err, &apiError)
			assert.Equal(t, status, apiError.StatusCode)
			assert.Equal(t, int32(1), calls.Load())
		})
	}
}

func TestPublicSandboxCreateReadiness(t *testing.T) {
	for _, tc := range []struct{ name, state, wantErr string }{
		{name: "pending reaches running", state: "RUNNING"},
		{name: "provisioning failure", state: "STOPPED", wantErr: `unexpected state "Stopped" while provisioning`},
		{name: "cancel pending create", state: "PENDING", wantErr: "context deadline exceeded"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gets := 0
			api := newPublicAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
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
				assert.Equal(t, "Running", got.Status)
			}
			assert.Equal(t, 1, gets)
			assert.Equal(t, []cachedSandbox{{ID: "test-id"}}, getSandboxes(ctx, api.c.Config.Host))
		})
	}
}

func TestPublicSandboxRejectsInvalidIDsBeforeRequests(t *testing.T) {
	for _, id := range []string{"", ".", "..", "a/b", `a\b`, "a?b", "a#b", "a%2fb", "a b"} {
		t.Run(id, func(t *testing.T) {
			api := newPublicAPI(t, "", func(w http.ResponseWriter, r *http.Request) { t.Error("invalid ID sent to service") })
			_, err := api.get(t.Context(), id)
			require.ErrorContains(t, err, "invalid sandbox ID")
			_, err = api.update(t.Context(), id, nil, nil, nil)
			require.ErrorContains(t, err, "invalid sandbox ID")
			_, err = api.stop(t.Context(), id)
			require.ErrorContains(t, err, "invalid sandbox ID")
			_, err = api.start(t.Context(), id)
			require.ErrorContains(t, err, "invalid sandbox ID")
			require.ErrorContains(t, api.delete(t.Context(), id), "invalid sandbox ID")
		})
	}
}

func TestPublicSandboxCreateFailureIncludesGeneratedID(t *testing.T) {
	var id string
	calls := 0
	api := newPublicAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
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
	api := newPublicAPI(t, "", func(w http.ResponseWriter, r *http.Request) {
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

func TestPublicSandboxSentinelCopyPreservesAccountAndOriginalConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/databricks-config" {
			_, _ = io.WriteString(w, `{"oidc_endpoint":"https://workspace.test/oidc"}`)
			return
		}
		assert.Equal(t, "/api/2.0/sandboxes/test-id", r.URL.Path)
		assert.Empty(t, r.Header.Get(auth.WorkspaceIDHeader))
		_, _ = io.WriteString(w, publicRunning)
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host: server.URL, Token: "test-token", AccountID: "test-account", WorkspaceID: profile.WorkspaceIDNone,
	})
	require.NoError(t, err)
	api, err := newSandboxAPI(w)
	require.NoError(t, err)
	assert.NotSame(t, w.Config, api.c.Config)
	assert.Equal(t, "test-account", api.c.Config.AccountID)
	assert.Empty(t, api.c.Config.WorkspaceID)
	_, err = api.get(t.Context(), "test-id")
	require.NoError(t, err)
	assert.Equal(t, profile.WorkspaceIDNone, w.Config.WorkspaceID)
	assert.Equal(t, "test-account", w.Config.AccountID)
	assert.Equal(t, server.URL, w.Config.Host)
	assert.Equal(t, "test-token", w.Config.Token)
}
