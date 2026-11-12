package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	loginTestService     = "system.ai.github"
	loginTestCredPath    = "/api/2.1/unity-catalog/mcp-services/system.ai.github/user-credentials"
	loginTestServicePath = "/api/2.1/unity-catalog/mcp-services/system.ai.github"
)

// loginTestServer answers the credential endpoints from a script: each GET returns the next
// entry of states, where "" means NOT_FOUND, and the last entry repeats. DELETE answers
// deleted (204) or not found (404).
type loginTestServer struct {
	mu      sync.Mutex
	states  []string
	reads   int
	deleted bool
	// signedIn is what DELETE answers: true for 200, false for 404.
	signedIn bool
}

func (s *loginTestServer) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rw.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == loginTestServicePath && r.Method == http.MethodGet:
		_, _ = io.WriteString(rw, `{"name":"mcp-services/system.ai.github","config":{"source_connection":{"options":{"oauth_provider":"OAUTH_PROVIDER_GITHUB_MCP"}}}}`)
	case r.URL.Path == loginTestCredPath && r.Method == http.MethodGet:
		state := s.states[min(s.reads, len(s.states)-1)]
		s.reads++
		if state == "" {
			http.Error(rw, `{"error_code":"NOT_FOUND","message":"Credential is not found"}`, http.StatusNotFound)
			return
		}
		body, _ := json.Marshal(catalog.McpServiceUserMappedCredential{
			ProvisioningInfo: &catalog.ProvisioningInfo{State: catalog.ProvisioningInfoState(state)},
		})
		_, _ = rw.Write(body)
	case r.URL.Path == loginTestCredPath && r.Method == http.MethodDelete:
		if !s.signedIn {
			http.Error(rw, `{"error_code":"NOT_FOUND","message":"Credential is not found"}`, http.StatusNotFound)
			return
		}
		s.deleted = true
		_, _ = io.WriteString(rw, `{}`)
	default:
		// The SDK resolves host metadata with requests of its own; those are not the endpoints under test.
		http.NotFound(rw, r)
	}
}

// loginTestClient starts the fake and returns a client for it.
func loginTestClient(t *testing.T, s *loginTestServer) *databricks.WorkspaceClient {
	t.Helper()
	server := httptest.NewServer(s)
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	return w
}

func TestSignInPollsUntilActive(t *testing.T) {
	s := &loginTestServer{states: []string{"", "PROVISIONING", "ACTIVE"}}
	w := loginTestClient(t, s)

	var opened []string
	var out bytes.Buffer
	err := SignIn(t.Context(), w, w.Config.Host, "123", loginTestService, &out, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
		Open: func(_ context.Context, url string) error {
			opened = append(opened, url)
			return nil
		},
	})
	require.NoError(t, err)

	wantURL := w.Config.Host + "/mcp-service-login?name=system.ai.github&o=123"
	assert.Equal(t, []string{wantURL}, opened)
	// The URL is printed before the browser is opened, and polling starts after that.
	assert.Contains(t, out.String(), "Open this URL to sign in to system.ai.github:\n"+wantURL+"\n")
	assert.Equal(t, 3, s.reads)
}

func TestSignInAlreadyActiveDoesNotOpenBrowser(t *testing.T) {
	s := &loginTestServer{states: []string{"ACTIVE"}}
	w := loginTestClient(t, s)

	var out bytes.Buffer
	err := SignIn(t.Context(), w, w.Config.Host, "123", loginTestService, &out, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: time.Second,
		Open: func(context.Context, string) error {
			t.Fatal("browser must not open when already signed in")
			return nil
		},
	})
	require.NoError(t, err)
	assert.Empty(t, out.String())
}

func TestSignInBrowserNoneOnlyPrintsURL(t *testing.T) {
	t.Setenv("BROWSER", "none")
	s := &loginTestServer{states: []string{"", "ACTIVE"}}
	w := loginTestClient(t, s)

	var out bytes.Buffer
	err := SignIn(t.Context(), w, w.Config.Host, "123", loginTestService, &out, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
		Open: func(context.Context, string) error {
			t.Fatal("browser must not open with BROWSER=none")
			return nil
		},
	})
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(out.String(), "/mcp-service-login?"), "the URL is printed once")
}

func TestSignInTimeout(t *testing.T) {
	s := &loginTestServer{states: []string{""}}
	w := loginTestClient(t, s)

	var out bytes.Buffer
	err := SignIn(t.Context(), w, w.Config.Host, "123", loginTestService, &out, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 30 * time.Millisecond,
	})
	require.ErrorIs(t, err, ErrSignInTimeout)
	assert.Contains(t, out.String(), "/mcp-service-login?name=system.ai.github&o=123")
}

func TestSignInCancelledIsNotATimeout(t *testing.T) {
	s := &loginTestServer{states: []string{""}}
	w := loginTestClient(t, s)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := SignIn(ctx, w, w.Config.Host, "123", loginTestService, io.Discard, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: time.Minute,
	})
	require.ErrorIs(t, err, context.Canceled)
	assert.NotErrorIs(t, err, ErrSignInTimeout)
}

func TestSignInFailed(t *testing.T) {
	s := &loginTestServer{states: []string{"", "FAILED"}}
	w := loginTestClient(t, s)

	err := SignIn(t.Context(), w, w.Config.Host, "123", loginTestService, io.Discard, SignInOptions{
		Poll:    time.Millisecond,
		Timeout: 10 * time.Second,
	})
	require.ErrorIs(t, err, ErrSignInFailed)
}

func TestCredentialStateNotSignedIn(t *testing.T) {
	s := &loginTestServer{states: []string{""}}
	w := loginTestClient(t, s)

	state, err := CredentialState(t.Context(), w, loginTestService)
	require.NoError(t, err)
	assert.Empty(t, state)
}

func TestSignOut(t *testing.T) {
	s := &loginTestServer{signedIn: true}
	w := loginTestClient(t, s)

	deleted, err := SignOut(t.Context(), w, loginTestService)
	require.NoError(t, err)
	assert.True(t, deleted)
	assert.True(t, s.deleted)

	// Once signed out, the server answers NOT_FOUND, which is not an error.
	s.signedIn = false
	deleted, err = SignOut(t.Context(), w, loginTestService)
	require.NoError(t, err)
	assert.False(t, deleted)
}

func TestGetServiceSignInRequired(t *testing.T) {
	s := &loginTestServer{}
	w := loginTestClient(t, s)

	svc, err := GetService(t.Context(), w, loginTestService)
	require.NoError(t, err)
	assert.True(t, svc.SignInRequired)
	assert.Equal(t, "OAUTH_PROVIDER_GITHUB_MCP", svc.OAuthProvider)
	require.NotNil(t, svc.Known)
	assert.Equal(t, "GitHub", svc.Known.Label)
}
