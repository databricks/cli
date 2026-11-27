package aitools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/proxy"
	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/databricks/cli/libs/auth/storage"
	"github.com/databricks/cli/libs/auth/u2m"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

// testService and testProfile are the service and the profile of the serve tests.
const (
	testService = "system.ai.github"
	testProfile = "logfood"
)

func TestMcpServeIsHidden(t *testing.T) {
	assert.True(t, NewMcpServeCmd().Hidden)
}

// TestServeParsesArgvThatAddWrites checks that the argv that `aitools mcp add` writes into an agent config parses with
// serve's flags.
func TestServeParsesArgvThatAddWrites(t *testing.T) {
	parent := newMCPTestRoot()
	serve, _, err := parent.Find([]string{"mcp", "serve"})
	require.NoError(t, err)
	argv := []string{mcpconfig.ProfileFlag, testProfile, mcpconfig.ServiceFlag, testService, mcpconfig.AgentFlag, "claude-code"}
	require.NoError(t, serve.ParseFlags(argv))

	profile, err := serve.Flags().GetString("profile")
	require.NoError(t, err)
	service, err := serve.Flags().GetString("service")
	require.NoError(t, err)
	agent, err := serve.Flags().GetString("agent")
	require.NoError(t, err)
	assert.Equal(t, []string{testProfile, testService, "claude-code"}, []string{profile, service, agent})
}

func TestRefreshGuardPassesThroughWhileTokensRefresh(t *testing.T) {
	cancelled := false
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return nil },
		profile:      testProfile,
		cancel:       func() { cancelled = true },
	}

	require.NoError(t, g.authenticateRequest(&http.Request{Header: http.Header{}}))
	assert.NoError(t, g.failure())
	assert.False(t, cancelled)
}

func TestRefreshGuardStopsProxyOnFirstFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	cause := refreshRejected
	g := &refreshGuard{
		authenticate: func(*http.Request) error {
			calls++
			return cause
		},
		profile: testProfile,
		cancel:  cancel,
	}

	err := g.authenticateRequest(&http.Request{Header: http.Header{}})
	require.Error(t, err)
	assert.ErrorIs(t, err, cause)
	assert.Contains(t, err.Error(), refreshLoginHint)
	assert.Equal(t, context.Canceled, ctx.Err())
	assert.ErrorIs(t, g.failure(), cause)
}

// TestRefreshGuardStopsLaterRequestsWithoutAskingAgain checks that after the first failure no request asks for a token
// again. Each later request fails at once with a short error that does not repeat the login hint.
func TestRefreshGuardStopsLaterRequestsWithoutAskingAgain(t *testing.T) {
	calls := 0
	cause := refreshRejected
	g := &refreshGuard{
		authenticate: func(*http.Request) error {
			calls++
			return cause
		},
		profile: testProfile,
		cancel:  func() {},
	}
	require.Error(t, g.authenticateRequest(&http.Request{Header: http.Header{}}))

	err := g.authenticateRequest(&http.Request{Header: http.Header{}})

	assert.Equal(t, 1, calls)
	assert.EqualError(t, err, "the session is stopping after a token refresh failure for profile \"logfood\"")
	assert.ErrorIs(t, g.failure(), cause)
}

// TestRefreshGuardNamesPinnedConfigFile checks that a refresh failure of a server pinned to a config file names that file
// in its login command, so that the command logs in to the file the profile is read from.
func TestRefreshGuardNamesPinnedConfigFile(t *testing.T) {
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return refreshRejected },
		profile:      testProfile,
		pinnedFile:   "/work/cfg/logfood.cfg",
		cancel:       func() {},
	}

	_ = g.authenticateRequest(&http.Request{Header: http.Header{}})

	require.Error(t, g.failure())
	assert.Contains(t, g.failure().Error(), "Run: DATABRICKS_CONFIG_FILE=/work/cfg/logfood.cfg databricks auth login --profile logfood")
}

// TestRefreshGuardStoresAuthError checks that the refresh failure reaches the user as the AUTH_FAILED error, which names
// the profile and its login command.
func TestRefreshGuardStoresAuthError(t *testing.T) {
	cause := refreshRejected
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return cause },
		profile:      "my profile",
		cancel:       func() {},
	}

	_ = g.authenticateRequest(&http.Request{Header: http.Header{}})

	be, ok := errors.AsType[*mcp.BindError](g.failure())
	require.True(t, ok)
	assert.Equal(t, mcp.BindAuthFailed, be.Category)
	assert.ErrorIs(t, be, cause)
	assert.Contains(t, be.Error(), "databricks auth login --profile 'my profile'")
}

// TestRefreshGuardAuthErrorNamesPinnedConfigFile checks that the login command for a profile pinned to a config file
// writes that file, as the sign-in redirect's hint does.
func TestRefreshGuardAuthErrorNamesPinnedConfigFile(t *testing.T) {
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return refreshRejected },
		profile:      "logfood",
		pinnedFile:   "/work/.databrickscfg",
		cancel:       func() {},
	}

	_ = g.authenticateRequest(&http.Request{Header: http.Header{}})

	assert.Contains(t, g.failure().Error(),
		"Run: DATABRICKS_CONFIG_FILE=/work/.databrickscfg databricks auth login --profile logfood\nThen restart the MCP server in your agent")
}

// TestRefreshGuardTransientFailureFailsOnlyThatRequest checks that a token failure which is not a rejection fails that
// request with the "try again" message and no login command. The proxy keeps running, and the next request asks for a
// token again.
func TestRefreshGuardTransientFailureFailsOnlyThatRequest(t *testing.T) {
	cancelled := false
	asked := 0
	transient := errors.New("dial tcp: i/o timeout")
	g := &refreshGuard{
		authenticate: func(*http.Request) error {
			asked++
			if asked == 1 {
				return transient
			}
			return nil
		},
		profile: "logfood",
		cancel:  func() { cancelled = true },
	}

	err := g.authenticateRequest(&http.Request{Header: http.Header{}})

	require.Error(t, err)
	assert.ErrorIs(t, err, transient)
	assert.Contains(t, err.Error(), `could not get a token for Databricks profile "logfood"`)
	assert.NotContains(t, err.Error(), "auth login", "a login does not fix a transient failure")
	assert.NotContains(t, err.Error(), "restart")
	assert.NoError(t, g.failure())
	assert.False(t, cancelled)
	assert.NoError(t, g.authenticateRequest(&http.Request{Header: http.Header{}}))
	assert.Equal(t, 2, asked)
}

// writeLogfoodProfile writes a config file whose logfood profile names host, and returns the error of the write.
func writeLogfoodProfile(path, host string) error {
	return os.WriteFile(path, []byte("[logfood]\nhost = "+host+"\ntoken = unused\n"), 0o600)
}

// boundServerState returns the state of a server bound to the logfood profile at host in the config file at path.
func boundServerState(host, path string) *mcp.State {
	return &mcp.State{Servers: []mcp.Entry{{Name: "github-logfood", Service: "system.ai.github", Profile: "logfood", Host: host, ConfigFile: path}}}
}

// TestRefreshGuardStopsSignInCallOnBindingChange checks that the sign-in client's requests are checked too: a profile
// re-pointed at another host fails the call before any token is fetched, and stops the proxy.
func TestRefreshGuardStopsSignInCallOnBindingChange(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	path := filepath.Join(t.TempDir(), "databrickscfg")
	require.NoError(t, writeLogfoodProfile(path, "https://b.test"))
	state := boundServerState("https://a.test", path)
	calls := 0
	g := &refreshGuard{
		authenticate: rebindingAuthenticate(bindingCheck(t.Context(), state, &state.Servers[0], "system.ai.github", "logfood", &config.Config{ConfigFile: path}),
			func(*http.Request) error {
				calls++
				return nil
			}),
		profile: "logfood",
		cancel:  cancel,
	}

	err := g.authenticateCall(&http.Request{Header: http.Header{}})

	be, ok := errors.AsType[*mcp.BindError](err)
	require.True(t, ok, err)
	assert.Equal(t, mcp.BindProfileHostChanged, be.Category)
	assert.Equal(t, context.Canceled, ctx.Err())
	assert.Zero(t, calls, "no token is fetched")
}

// TestRefreshGuardStopsOnBindingChange checks that a profile re-pointed at another host fails the request with the
// binding error, before the token is fetched, and stops the proxy as a rejected token does. Later requests fail at once.
func TestRefreshGuardStopsOnBindingChange(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	path := filepath.Join(t.TempDir(), "databrickscfg")
	require.NoError(t, writeLogfoodProfile(path, "https://b.test"))
	calls := 0
	state := boundServerState("https://a.test", path)
	g := &refreshGuard{
		authenticate: rebindingAuthenticate(bindingCheck(t.Context(), state, &state.Servers[0], "system.ai.github", "logfood", &config.Config{ConfigFile: path}),
			func(*http.Request) error {
				calls++
				return nil
			}),
		profile: "logfood",
		cancel:  cancel,
	}

	err := g.authenticateRequest(&http.Request{Header: http.Header{}})

	be, ok := errors.AsType[*mcp.BindError](err)
	require.True(t, ok, err)
	assert.Equal(t, mcp.BindProfileHostChanged, be.Category)
	assert.Contains(t, err.Error(), `profile "logfood" now points at https://b.test`)
	assert.Equal(t, context.Canceled, ctx.Err())
	assert.EqualError(t, g.authenticateRequest(&http.Request{Header: http.Header{}}),
		`the session is stopping after the binding changed for profile "logfood"`)
	assert.Equal(t, 0, calls, "no token is fetched once the binding has changed")
}

// TestBindingCheckWithoutRecordChecksStartHost checks that with no install record, the host serve bound to at start is
// the one the per-token check holds the profile to, and that the error names no record to remove.
func TestBindingCheckWithoutRecordChecksStartHost(t *testing.T) {
	path := filepath.Join(t.TempDir(), "databrickscfg")
	require.NoError(t, writeLogfoodProfile(path, "https://a.test"))
	cfg := &config.Config{Profile: "logfood", ConfigFile: path, Host: "https://a.test"}
	check := bindingCheck(t.Context(), &mcp.State{}, nil, "system.ai.github", "logfood", cfg)

	require.NoError(t, check())

	require.NoError(t, writeLogfoodProfile(path, "https://b.test"))
	err := check()
	assert.ErrorContains(t, err, `profile "logfood" now points at https://b.test`)
	assert.NotContains(t, err.Error(), "aitools mcp remove")
}

// TestServeProxyRepointedProfileSendsNoToken checks that a profile re-pointed at another host while serve runs gets no
// token sent to the host serve is bound to. The re-point happens while the first tools/list is answered, so the next
// request fails with the binding error before its token is fetched. The proxy stops with exit 2, and no request reaches
// the server after the re-point.
func TestServeProxyRepointedProfileSendsNoToken(t *testing.T) {
	const cachedToken = "Bearer token-cached-for-logfood"
	path := filepath.Join(t.TempDir(), "databrickscfg")
	var mu sync.Mutex
	var received int
	var repointedAt int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		received++
		mu.Unlock()
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if msg.Method == "tools/list" {
			mu.Lock()
			repointedAt = received
			mu.Unlock()
			assert.NoError(t, writeLogfoodProfile(path, "https://other.test"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "session-1")
		result := `{"tools":[]}`
		if msg.Method == "initialize" {
			result = `{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"fake","version":"1"}}`
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, msg.ID, result)
	}))
	t.Cleanup(server.Close)
	require.NoError(t, writeLogfoodProfile(path, server.URL))
	state := boundServerState(server.URL, path)
	authenticate := rebindingAuthenticate(bindingCheck(t.Context(), state, &state.Servers[0], "system.ai.github", "logfood", &config.Config{ConfigFile: path}),
		func(r *http.Request) error {
			r.Header.Set("Authorization", cachedToken)
			return nil
		})

	var stdout lockedBuffer
	var stderr bytes.Buffer
	in := feedAfterAnswers(t, &stdout, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
	}, nil)
	cmd := NewMcpServeCmd()
	cmd.SetContext(t.Context())
	cmd.SetErr(&stderr)

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	refresh := &refreshGuard{authenticate: authenticate, profile: "logfood", cancel: stop}
	err := serveProxy(ctx, cmd, refresh, proxy.Options{
		In:           in,
		Out:          &stdout,
		Err:          &stderr,
		Endpoint:     mcp.ServiceURL(server.URL, "system.ai.github"),
		Config:       &config.Config{Host: server.URL},
		Authenticate: refresh.authenticateRequest,
		Profile:      "logfood",
	})

	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
	assert.Contains(t, stdout.String(), `"id":3,"error"`)
	assert.Contains(t, stdout.String(), "now points at https://other.test")
	assert.True(t, strings.HasSuffix(stderr.String(), "Error: stopped after the binding changed for profile \"logfood\"\n"), stderr.String())
	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, 2, repointedAt, "the server answers initialize and the first tools/list before the re-point")
	assert.Equal(t, repointedAt, received, "no request reaches the server after the re-point")
}

// TestCheckBindingWithRetry checks that a changed binding is checked once more, since a write that replaces
// .databrickscfg can hide the profile for a moment. A failure that is not a changed binding is not checked again.
func TestCheckBindingWithRetry(t *testing.T) {
	notFound := &mcp.BindError{Category: mcp.BindProfileNotFound}
	other := errors.New("dial tcp: i/o timeout")
	tests := []struct {
		name      string
		results   []error // the result of each call; the last one repeats
		wantErr   error
		wantCalls int
	}{
		{name: "passes at once", results: []error{nil}, wantErr: nil, wantCalls: 1},
		{name: "changed once, then passes", results: []error{notFound, nil}, wantErr: nil, wantCalls: 2},
		{name: "still changed", results: []error{notFound, notFound}, wantErr: notFound, wantCalls: 2},
		{name: "not a changed binding", results: []error{other, nil}, wantErr: other, wantCalls: 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			err := checkBindingWithRetry(func() error {
				calls++
				return tc.results[min(calls, len(tc.results))-1]
			})

			assert.Equal(t, tc.wantCalls, calls)
			if tc.wantErr == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

// oauthResponse is an error response from the CLI's U2M client, which returns an HttpError for a non-2xx answer.
func oauthResponse(status int, body string) error {
	return &httpclient.HttpError{Response: &http.Response{StatusCode: status}, Message: body}
}

// TestCredentialRejected checks which token failures stop the proxy. A rejection is one that a new login fixes. Every
// other failure is transient, including the ones this check cannot classify.
func TestCredentialRejected(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "refresh rejected, golang.org/x/oauth2 error", err: refreshRejected, want: true},
		{name: "invalid_client with 401", err: &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusUnauthorized}, ErrorCode: "invalid_client"}, want: true},
		{name: "invalid_grant with 500", err: &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusInternalServerError}, ErrorCode: "invalid_grant"}, want: false},
		{name: "server_error with 400", err: &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusBadRequest}, ErrorCode: "server_error"}, want: false},
		{name: "oauth2 response without an error code", err: &oauth2.RetrieveError{Response: &http.Response{StatusCode: http.StatusBadRequest}, Body: []byte("<html>")}, want: false},
		{name: "HTTP invalid_grant with 400", err: oauthResponse(400, `{"error":"invalid_grant","error_description":"Refresh token has expired"}`), want: true},
		{name: "HTTP invalid_client with 401", err: oauthResponse(401, `{"error":"invalid_client"}`), want: true},
		{name: "HTTP invalid_grant, wrapped", err: fmt.Errorf("token refresh: %w", oauthResponse(400, `{"error":"invalid_grant"}`)), want: true},
		{name: "HTTP invalid_grant with 503", err: oauthResponse(503, `{"error":"invalid_grant"}`), want: false},
		{name: "HTTP invalid_request with 400", err: oauthResponse(400, `{"error":"invalid_request"}`), want: false},
		{name: "HTTP body is not JSON", err: oauthResponse(400, "Bad Gateway"), want: false},
		{name: "network error", err: errors.New("dial tcp: i/o timeout"), want: false},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: false},
		{name: "no refresh token", err: fmt.Errorf("token refresh: %w", u2m.ErrMissingRefreshToken), want: true},
		{name: "no cached token after logout", err: fmt.Errorf("cache: %w", storage.ErrNotFound), want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, credentialRejected(tc.err))
		})
	}
}

func TestServeFailurePrintsOnceWithStatus(t *testing.T) {
	cmd := NewMcpServeCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cause := mcp.AuthError(testProfile, errors.New(refreshTokenExpired))

	err := serveFailure(cmd, cause, serveBindStatus)

	assert.Equal(t, "Error: "+cause.Error()+"\n", stderr.String())
	assert.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
	assert.ErrorIs(t, err, cause)
}

// TestServeBindFailureTokenStatus checks that a token failure at startup is classified as it is at runtime. A
// rejection needs a new login, so it keeps the login hint and exits 2. Any other failure is temporary: it has no hint
// and exits 1, the status of the other failures, since exit 2 says the credential was rejected.
func TestServeBindFailureTokenStatus(t *testing.T) {
	loginHint := mcp.LoginCommand("logfood", "")
	tests := []struct {
		name   string
		cause  error
		status int
		hint   bool
	}{
		{name: "rejected refresh token", cause: refreshRejected, status: 2, hint: true},
		{name: "no cached token", cause: fmt.Errorf("cache: %w", storage.ErrNotFound), status: 2, hint: true},
		{name: "no strategy configures a credential", cause: config.ErrCannotConfigureDefault, status: 2, hint: true},
		{name: "network error", cause: errors.New("dial tcp: i/o timeout"), status: 1, hint: false},
		{name: "server error from the token endpoint", cause: oauthResponse(503, `{"error":"server_error"}`), status: 1, hint: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := NewMcpServeCmd()
			var stderr bytes.Buffer
			cmd.SetErr(&stderr)

			err := serveBindFailure(cmd, "logfood", mcp.AuthErrorIn("logfood", "", tc.cause))

			assert.ErrorIs(t, err, root.ErrAlreadyPrinted)
			assert.Equal(t, tc.status, root.ExitStatus(err))
			assert.Equal(t, tc.hint, strings.Contains(stderr.String(), loginHint))
			if tc.status == 1 {
				assert.Contains(t, stderr.String(), "temporary failure")
			}
		})
	}
}

// TestServeBindHostOnlyProfileNeedsLogin runs the real credential chain on a profile with a host and no auth_type, as a
// profile written before auth login set one, with no cached token. The chain drops each strategy's error and returns
// config.ErrCannotConfigureDefault, which a login fixes, so serve exits 2 with the login hint.
func TestServeBindHostOnlyProfileNeedsLogin(t *testing.T) {
	// The workspace is a local server that knows no endpoint, so the host-metadata lookup and the OAuth discovery of
	// the chain stay in the test binary.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	home := withHome(t)
	writeHomeConfig(t, home, "[legacy]\nhost = "+server.URL+"\n")

	_, _, err := mcp.Bind(t.Context(), mcp.BindOptions{
		Profile: "legacy",
		Service: "system.ai.github",
		State:   &mcp.State{},
		Stderr:  &bytes.Buffer{},
	})
	require.ErrorIs(t, err, config.ErrCannotConfigureDefault)

	cmd := NewMcpServeCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	serveErr := serveBindFailure(cmd, "legacy", err)

	assert.Equal(t, serveBindStatus, root.ExitStatus(serveErr))
	assert.Contains(t, stderr.String(), "\nRun: "+mcp.LoginCommand("legacy", ""))
	assert.NotContains(t, stderr.String(), "temporary failure")
}

// TestServeTransportTLSVerification checks that the profile's insecure_skip_verify reaches the transport, and that
// HTTP/2 is kept with it.
func TestServeTransportTLSVerification(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *config.Config
		insecure bool
	}{
		{name: "verifies by default", cfg: &config.Config{}, insecure: false},
		{name: "profile skips verification", cfg: &config.Config{InsecureSkipVerify: true}, insecure: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tr := serveTransport(tc.cfg)
			assert.Equal(t, tc.insecure, tr.TLSClientConfig != nil && tr.TLSClientConfig.InsecureSkipVerify)
			assert.True(t, tr.ForceAttemptHTTP2)
		})
	}
}

func TestServeTransportKeepsHTTP2WhenSkippingVerification(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	server.EnableHTTP2 = true
	server.StartTLS()
	defer server.Close()

	client := &http.Client{Transport: serveTransport(&config.Config{InsecureSkipVerify: true})}
	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, 2, resp.ProtoMajor)
}

// TestRefreshGuardIgnoresRequestThatEnded checks that a request whose own context ended (the client cancelled it, or
// shutdown began) fails that request only: the token was not shown to be unusable.
func TestRefreshGuardIgnoresRequestThatEnded(t *testing.T) {
	cancelled := false
	g := &refreshGuard{
		authenticate: func(r *http.Request) error { return r.Context().Err() },
		profile:      testProfile,
		cancel:       func() { cancelled = true },
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := g.authenticateRequest((&http.Request{Header: http.Header{}}).WithContext(ctx))

	assert.ErrorIs(t, err, context.Canceled)
	assert.NoError(t, g.failure())
	assert.False(t, cancelled)
}

// TestRefreshGuardIgnoresSessionDelete checks that the session DELETE at shutdown is best effort. Its auth failure is
// logged by the bridge, and a normal shutdown exits 0.
func TestRefreshGuardIgnoresSessionDelete(t *testing.T) {
	cancelled := false
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return refreshRejected },
		profile:      testProfile,
		cancel:       func() { cancelled = true },
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "https://workspace.test/mcp", nil)
	require.NoError(t, err)

	require.Error(t, g.authenticateRequest(req))

	assert.NoError(t, g.failure())
	assert.False(t, cancelled)
}

// TestServeAgentKeepsOnlyKnownAgents checks that an unknown agent is left out of the User-Agent and reported once on
// stderr, so the server still serves.
func TestServeAgentKeepsOnlyKnownAgents(t *testing.T) {
	var stderr bytes.Buffer
	assert.Equal(t, "claude-code", serveAgent("claude-code", &stderr))
	assert.Empty(t, stderr.String())

	assert.Empty(t, serveAgent("agent-from-a-later-release", &stderr))
	assert.Equal(t, "unknown agent \"agent-from-a-later-release\"; it is left out of the User-Agent\n", stderr.String())

	stderr.Reset()
	assert.Empty(t, serveAgent("", &stderr))
	assert.Empty(t, stderr.String())
}

// TestMcpServeRejectsServiceWithoutThreeParts checks that a service that is not a full three-part name is refused before
// any binding check, with the flag named.
func TestMcpServeRejectsServiceWithoutThreeParts(t *testing.T) {
	withHome(t)
	for _, service := range []string{"github", "system.ai", "system..github", "a.b.c.d", "system.ai.git/hub"} {
		t.Run(service, func(t *testing.T) {
			parent := newMCPTestRoot()
			parent.SetArgs([]string{"mcp", "serve", "--profile", testProfile, "--service", service})
			err := parent.ExecuteContext(cmdio.MockDiscard(t.Context()))
			require.ErrorContains(t, err, "--service needs a full service name such as system.ai.github")
			assert.Equal(t, 1, root.ExitStatus(err))
		})
	}
}

// TestMcpServeFailedStartLogsNothing checks that a start that fails its binding checks logs no PROXY_STARTED, so nothing
// uploads.
func TestMcpServeFailedStartLogsNothing(t *testing.T) {
	withHome(t)
	parent := newMCPTestRoot()
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs([]string{"mcp", "serve", "--profile", testProfile, "--service", testService})
	ctx := env.Set(telemetry.WithNewLogger(cmdio.MockDiscard(t.Context())), "DATABRICKS_CLI_DISABLE_TELEMETRY", "")
	err := parent.ExecuteContext(ctx)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, serveBindStatus, root.ExitStatus(err))
	serve, _, findErr := parent.Find([]string{"mcp", "serve"})
	require.NoError(t, findErr)
	assert.Empty(t, telemetry.BufferedEvents(serve.Context()))
}

// TestMcpServeRejectedStartLogsNothing checks that a start whose startup read is rejected by the profile's credential logs
// no PROXY_STARTED, so the run has nothing to upload, and sets no upload config.
func TestMcpServeRejectedStartLogsNothing(t *testing.T) {
	home := withHome(t)
	server := serviceServer(t, http.StatusUnauthorized, `{"error_code":"UNAUTHENTICATED","message":"Invalid access token."}`)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nauth_type = pat\ntoken = dapi-expired\n")

	parent := newMCPTestRoot()
	parent.SetErr(&bytes.Buffer{})
	parent.SetArgs([]string{"mcp", "serve", "--profile", testProfile, "--service", testService})
	ctx := telemetry.WithNewLogger(cmdio.MockDiscard(t.Context()))
	err := parent.ExecuteContext(ctx)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, serveBindStatus, root.ExitStatus(err))
	serve, _, findErr := parent.Find([]string{"mcp", "serve"})
	require.NoError(t, findErr)
	assert.Empty(t, telemetry.BufferedEvents(serve.Context()))
	_, uploads := telemetry.UploadConfig(serve.Context())
	assert.False(t, uploads)
}

// TestMcpServeUploadUsesProfileBoundAtStart checks that the exit upload of a serving run authenticates with the profile as
// it was bound at start, so a token that the config file gets during the session does not change the upload.
func TestMcpServeUploadUsesProfileBoundAtStart(t *testing.T) {
	home := withHome(t)
	server := newEchoMCPServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-bound\n")

	parent := newMCPTestRoot()
	parent.SetErr(&bytes.Buffer{})
	var stdout lockedBuffer
	parent.SetOut(&stdout)
	in, stdin := io.Pipe()
	parent.SetIn(in)
	parent.SetArgs([]string{"mcp", "serve", "--profile", testProfile, "--service", testService})
	ctx := serveRunContext(t)
	done := make(chan error, 1)
	go func() { done <- parent.ExecuteContext(ctx) }()

	_, err := io.WriteString(stdin, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n")
	require.NoError(t, err)
	require.Eventually(t, func() bool { return strings.Contains(stdout.String(), `"id":1`) }, 10*time.Second, time.Millisecond)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-rotated\n")
	require.NoError(t, stdin.Close())
	require.NoError(t, <-done)

	serve, _, findErr := parent.Find([]string{"mcp", "serve"})
	require.NoError(t, findErr)
	cfg, uploads := telemetry.UploadConfig(serve.Context())
	require.True(t, uploads)
	assert.Equal(t, "dapi-bound", cfg.Token)
}

// serveRunContext returns the context of a serve run whose telemetry uploads: enabled, with the exec ID and runtime that
// cmd/root sets on a run.
func serveRunContext(t *testing.T) context.Context {
	t.Helper()
	ctx := env.Set(telemetry.WithNewLogger(cmdio.MockDiscard(t.Context())), "DATABRICKS_CLI_DISABLE_TELEMETRY", "")
	return cmdctx.GenerateExecId(dbr.DetectRuntime(ctx))
}

// TestMcpServeUploadsProxyStartedAtStart checks that serve uploads its PROXY_STARTED while it still serves, so a serve that
// is killed, or whose token expires while it idles, has sent it. The exit upload does not send it again.
func TestMcpServeUploadsProxyStartedAtStart(t *testing.T) {
	for _, authType := range []string{"pat", "databricks-cli"} {
		t.Run(authType, func(t *testing.T) {
			testMcpServeUploadsProxyStartedAtStart(t, authType)
		})
	}
}

func testMcpServeUploadsProxyStartedAtStart(t *testing.T, authType string) {
	t.Helper()
	home := withHome(t)
	const token = "test-bound-token"
	server, uploaded := newEchoMCPServerWithUploads(t, "Bearer "+token)
	profile := "[logfood]\nhost = " + server.URL + "\nworkspace_id = 222\nauth_type = " + authType + "\n"
	if authType == "databricks-cli" {
		t.Setenv(storage.EnvVar, string(storage.StorageModePlaintext))
		store, err := storage.NewFileStore(t.Context())
		require.NoError(t, err)
		require.NoError(t, store.Put(testProfile, storage.Entry{
			Token: &oauth2.Token{AccessToken: token, Expiry: time.Now().Add(time.Hour)},
		}))
	} else {
		profile += "token = " + token + "\n"
	}
	writeHomeConfig(t, home, profile)

	parent := newMCPTestRoot()
	parent.SetErr(&bytes.Buffer{})
	var stdout lockedBuffer
	parent.SetOut(&stdout)
	in, stdin := io.Pipe()
	t.Cleanup(func() { stdin.Close() })
	parent.SetIn(in)
	parent.SetArgs([]string{"mcp", "serve", "--profile", testProfile, "--service", testService})
	ctx := serveRunContext(t)
	done := make(chan error, 1)
	go func() { done <- parent.ExecuteContext(ctx) }()

	_, err := io.WriteString(stdin, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n")
	require.NoError(t, err)
	require.Eventually(t, func() bool { return strings.Contains(stdout.String(), `"id":1`) }, 10*time.Second, time.Millisecond)
	assert.Equal(t, []protos.AitoolsMcpOperation{protos.AitoolsMcpOperationProxyStarted}, uploaded(), "serve is still running")

	require.NoError(t, stdin.Close())
	require.NoError(t, <-done)
	serve, _, findErr := parent.Find([]string{"mcp", "serve"})
	require.NoError(t, findErr)
	require.NoError(t, telemetry.Upload(serve.Context(), protos.ExecutionContext{}))
	assert.Equal(t, []protos.AitoolsMcpOperation{protos.AitoolsMcpOperationProxyStarted}, uploaded(), "the exit upload does not resend it")
}

// TestMcpServeDropsUploadAfterRepoint checks that a profile re-pointed at another host during the session leaves serve
// with no upload config at exit, even when no request ran after the re-point.
func TestMcpServeDropsUploadAfterRepoint(t *testing.T) {
	home := withHome(t)
	server := newEchoMCPServer(t)
	writeHomeConfig(t, home, "[logfood]\nhost = "+server.URL+"\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-bound\n")

	parent := newMCPTestRoot()
	parent.SetErr(&bytes.Buffer{})
	var stdout lockedBuffer
	parent.SetOut(&stdout)
	in, stdin := io.Pipe()
	parent.SetIn(in)
	parent.SetArgs([]string{"mcp", "serve", "--profile", testProfile, "--service", testService})
	ctx := serveRunContext(t)
	done := make(chan error, 1)
	go func() { done <- parent.ExecuteContext(ctx) }()

	_, err := io.WriteString(stdin, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`+"\n")
	require.NoError(t, err)
	require.Eventually(t, func() bool { return strings.Contains(stdout.String(), `"id":1`) }, 10*time.Second, time.Millisecond)
	writeHomeConfig(t, home, "[logfood]\nhost = https://other.test\nworkspace_id = 222\nauth_type = pat\ntoken = dapi-other\n")
	require.NoError(t, stdin.Close())
	require.NoError(t, <-done)

	serve, _, findErr := parent.Find([]string{"mcp", "serve"})
	require.NoError(t, findErr)
	_, uploads := telemetry.UploadConfig(serve.Context())
	assert.False(t, uploads)
}

// TestMcpServeStateReadFailureExitsTwo checks that a state file that cannot be read stops the binding checks, so it
// exits 2 like the other binding failures.
func TestMcpServeStateReadFailureExitsTwo(t *testing.T) {
	withHome(t)
	dir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".state.json"), 0o700))

	parent := newMCPTestRoot()
	var stderr bytes.Buffer
	parent.SetErr(&stderr)
	parent.SetArgs([]string{"mcp", "serve", "--profile", testProfile, "--service", testService})
	err = parent.ExecuteContext(cmdio.MockDiscard(t.Context()))
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
	assert.Contains(t, stderr.String(), ".state.json")
}

// TestRefreshGuardSkipsSessionDeleteAfterFailure checks that after a refresh failure the session DELETE is skipped: the
// session is over, and a DELETE without a credential would fail and log the refresh error a second time. Authenticate is
// not asked.
func TestRefreshGuardSkipsSessionDeleteAfterFailure(t *testing.T) {
	asked := 0
	g := &refreshGuard{
		authenticate: func(*http.Request) error {
			asked++
			return refreshRejected
		},
		profile: testProfile,
		cancel:  func() {},
	}
	require.Error(t, g.authenticateRequest(&http.Request{Header: http.Header{}}))
	asked = 0

	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "https://workspace.test/mcp", nil)
	require.NoError(t, err)
	assert.ErrorIs(t, g.authenticateRequest(req), proxy.ErrSkipRequest)
	assert.Equal(t, 0, asked)
}

// TestProxyResultRefreshFailureWinsOverStdinError checks that once the bridge has returned, a refresh failure is the exit
// status and the final line, whatever Run returned. A stdin error that follows the failure is a consequence of it and is
// not reported.
func TestProxyResultRefreshFailureWinsOverStdinError(t *testing.T) {
	cause := refreshRejected
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return cause },
		profile:      testProfile,
		cancel:       func() {},
	}
	require.Error(t, g.authenticateRequest(&http.Request{Header: http.Header{}}))
	cmd := NewMcpServeCmd()
	cmd.SetContext(telemetry.WithNewLogger(t.Context()))
	telemetry.SetUploadConfig(cmd.Context(), &config.Config{Host: "https://adb-1.test"})
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := proxyResult(cmd, g, errors.New("read stdin: input/output error"))

	// The run uploads like any other: its upload config is the profile bound at start, which never refreshes a token.
	_, uploads := telemetry.UploadConfig(cmd.Context())
	assert.True(t, uploads)
	assert.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, serveBindStatus, root.ExitStatus(err))
	assert.ErrorIs(t, err, cause)
	assert.Equal(t, refreshStoppedLine, stderr.String())
}

// TestProxyResultReportsStdinErrorWithoutRefreshFailure checks that without a refresh failure, a stdin error is printed
// and exits 1.
func TestProxyResultReportsStdinErrorWithoutRefreshFailure(t *testing.T) {
	g := &refreshGuard{profile: testProfile}
	cmd := NewMcpServeCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := proxyResult(cmd, g, errors.New("read stdin: input/output error"))

	assert.Equal(t, 1, root.ExitStatus(err))
	assert.Equal(t, "Error: read stdin: input/output error\n", stderr.String())
}

// The refresh failure fixtures: the cause that the fake token gives, the login hint that serve prints with it, and the
// final line that serve prints once the proxy has stopped.
const (
	refreshTokenExpired = "refresh token expired"
	refreshLoginHint    = "databricks auth login --profile logfood\nThen restart the MCP server in your agent"
	refreshStoppedLine  = "Error: stopped after a token refresh failure for profile \"logfood\"\n"
)

// refreshRejected is the error of a refresh that the token endpoint rejects, which a new login fixes. It is the shape
// the SDK's client returns for an error response from golang.org/x/oauth2.
var refreshRejected = &oauth2.RetrieveError{
	Response:         &http.Response{StatusCode: http.StatusBadRequest},
	ErrorCode:        "invalid_grant",
	ErrorDescription: refreshTokenExpired,
}

// isToolsList reports whether r is a tools/list POST. It reads a copy of the body, so it is for Authenticate only.
func isToolsList(r *http.Request) (bool, error) {
	if r.Method != http.MethodPost {
		return false, nil
	}
	body, err := r.GetBody()
	if err != nil {
		return false, err
	}
	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), `"tools/list"`), nil
}

// expiresOnToolsList is an Authenticate whose token expires with the tools/list request, as a refresh token can
// mid-session.
func expiresOnToolsList(r *http.Request) error {
	listed, err := isToolsList(r)
	if err != nil || !listed {
		return err
	}
	return refreshRejected
}

// failsToolsListOnce is an Authenticate whose first tools/list fails with err. Later calls succeed, as they do once the
// token endpoint is back.
func failsToolsListOnce(err error) func(*http.Request) error {
	var failed atomic.Bool
	return func(r *http.Request) error {
		listed, readErr := isToolsList(r)
		if readErr != nil {
			return readErr
		}
		if listed && failed.CompareAndSwap(false, true) {
			return err
		}
		return nil
	}
}

// newEchoMCPServer is a fake endpoint that answers each request with its own id. Its initialize answer sets a session ID,
// so the session DELETE at shutdown is sent.
func newEchoMCPServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(echoMCP))
	t.Cleanup(server.Close)
	return server
}

// echoMCP answers an MCP request as newEchoMCPServer does.
func echoMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		w.WriteHeader(http.StatusOK)
		return
	}
	var msg struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Mcp-Session-Id", "session-1")
	result := `{"tools":[]}`
	if msg.Method == "initialize" {
		result = `{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"fake","version":"1"}}`
	}
	fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, msg.ID, result)
}

// newEchoMCPServerWithUploads is newEchoMCPServer that also takes telemetry uploads. The function it returns gives the
// operation of each AitoolsMcpEvent the uploads carried so far, in order.
func newEchoMCPServerWithUploads(t *testing.T, wantAuthorization string) (*httptest.Server, func() []protos.AitoolsMcpOperation) {
	t.Helper()
	var mu sync.Mutex
	var operations []protos.AitoolsMcpOperation
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/telemetry-ext" {
			echoMCP(w, r)
			return
		}
		assert.Equal(t, wantAuthorization, r.Header.Get("Authorization"))
		var body telemetry.RequestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for _, p := range body.ProtoLogs {
			var entry protos.FrontendLog
			if err := json.Unmarshal([]byte(p), &entry); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if e := entry.Entry.DatabricksCliLog.AitoolsMcpEvent; e != nil {
				operations = append(operations, e.Operation)
			}
		}
		fmt.Fprintf(w, `{"numProtoSuccess":%d}`, len(body.ProtoLogs))
	}))
	t.Cleanup(server.Close)
	return server, func() []protos.AitoolsMcpOperation {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(operations)
	}
}

// TestServeProxyTransientRefreshFailureKeepsServing checks that a token failure which is not a rejection fails only the
// request that needed the token. The agent gets the error, without a login hint. Serve keeps running, the next request
// asks for a token again and succeeds, and stdin closing exits 0.
func TestServeProxyTransientRefreshFailureKeepsServing(t *testing.T) {
	server := newEchoMCPServer(t)
	var stdout lockedBuffer
	var stderr bytes.Buffer
	in := feedAfterAnswers(t, &stdout, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
	}, nil)
	cmd := NewMcpServeCmd()
	cmd.SetContext(t.Context())
	cmd.SetErr(&stderr)

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	refresh := &refreshGuard{authenticate: failsToolsListOnce(errors.New("dial tcp: i/o timeout")), profile: "logfood", cancel: stop}
	err := serveProxy(ctx, cmd, refresh, proxy.Options{
		In:           in,
		Out:          &stdout,
		Err:          &stderr,
		Endpoint:     mcp.ServiceURL(server.URL, "system.ai.github"),
		Config:       &config.Config{Host: server.URL},
		Authenticate: refresh.authenticateRequest,
		Profile:      "logfood",
	})

	require.NoError(t, err)
	out := stdout.String()
	assert.Contains(t, out, `"id":2,"error"`)
	assert.Contains(t, out, "the next request tries again")
	assert.Contains(t, out, `"id":3,"result"`)
	assert.NotContains(t, stderr.String(), "stopped after")
}

// TestServeProxyRefreshRejectionStopsAfterTransientFailure checks that a rejection after a transient failure still stops
// the proxy: the earlier failure does not change how the rejection is handled.
func TestServeProxyRefreshRejectionStopsAfterTransientFailure(t *testing.T) {
	server := newEchoMCPServer(t)
	var stdout lockedBuffer
	var stderr bytes.Buffer
	in := feedAfterAnswers(t, &stdout, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
	}, nil)
	cmd := NewMcpServeCmd()
	cmd.SetContext(t.Context())
	cmd.SetErr(&stderr)
	transient := failsToolsListOnce(errors.New("dial tcp: i/o timeout"))
	rejected := func(r *http.Request) error {
		if err := transient(r); err != nil {
			return err
		}
		return expiresOnToolsList(r)
	}

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	refresh := &refreshGuard{authenticate: rejected, profile: "logfood", cancel: stop}
	err := serveProxy(ctx, cmd, refresh, proxy.Options{
		In:           in,
		Out:          &stdout,
		Err:          &stderr,
		Endpoint:     mcp.ServiceURL(server.URL, "system.ai.github"),
		Config:       &config.Config{Host: server.URL},
		Authenticate: refresh.authenticateRequest,
		Profile:      "logfood",
	})

	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
	assert.True(t, strings.HasSuffix(stderr.String(), refreshStoppedLine), stderr.String())
}

// newRotatingSessionServer is a fake endpoint for the debug log. Its initialize answer starts session sess-alpha. A
// request that carries sess-alpha is answered with sess-beta, a new session the server then expects.
func newRotatingSessionServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		result := `{"tools":[]}`
		switch {
		case msg.Method == "initialize":
			w.Header().Set("Mcp-Session-Id", "sess-alpha")
			result = `{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"fake","version":"1"}}`
		case r.Header.Get("Mcp-Session-Id") == "sess-alpha":
			w.Header().Set("Mcp-Session-Id", "sess-beta")
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, msg.ID, result)
	}))
	t.Cleanup(server.Close)
	return server
}

// TestServeLogsEachUpstreamRequestAtDebug checks that at debug level serve logs one line per upstream request: the
// JSON-RPC method and id, the HTTP status, the duration, and whether a session id was sent, received, or replaced. The
// lines carry no body and no header value, so neither the token nor a session id nor a tool argument appears in them.
func TestServeLogsEachUpstreamRequestAtDebug(t *testing.T) {
	server := newRotatingSessionServer(t)
	var stdout lockedBuffer
	var stderr bytes.Buffer
	var logs lockedBuffer
	in := feedAfterAnswers(t, &stdout, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{"cursor":"secret-cursor"}}`,
	}, nil)
	ctx := log.NewContext(t.Context(), slog.New(slog.NewTextHandler(&logs, &slog.HandlerOptions{Level: slog.LevelDebug})))
	cmd := NewMcpServeCmd()
	cmd.SetContext(ctx)
	cmd.SetErr(&stderr)

	ctx, stop := context.WithCancel(ctx)
	defer stop()
	refresh := &refreshGuard{
		authenticate: func(r *http.Request) error {
			r.Header.Set("Authorization", "Bearer secret-token")
			return nil
		},
		profile: "logfood",
		cancel:  stop,
	}
	err := serveProxy(ctx, cmd, refresh, proxy.Options{
		In:        in,
		Out:       &stdout,
		Err:       &stderr,
		Endpoint:  mcp.ServiceURL(server.URL, "system.ai.github"),
		Config:    &config.Config{Host: server.URL},
		Profile:   "logfood",
		Transport: newUpstreamLog(http.DefaultTransport),
	})

	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	require.Len(t, lines, 3, logs.String())
	// The initialize carries no session id and the server starts one; tools/list sends it and the server replaces it;
	// the shutdown DELETE sends the replacement.
	assert.Contains(t, lines[0], `msg="upstream request" http_method=POST rpc_method=initialize rpc_id=1 status=200`)
	assert.Contains(t, lines[0], "session_sent=false session_received=true session_replaced=false")
	assert.Contains(t, lines[1], `http_method=POST rpc_method=tools/list rpc_id=2 status=200`)
	assert.Contains(t, lines[1], "session_sent=true session_received=true session_replaced=true")
	assert.Contains(t, lines[2], `http_method=DELETE rpc_method="" rpc_id="" status=200`)
	assert.Contains(t, lines[2], "session_sent=true session_received=false session_replaced=false")
	assert.Contains(t, lines[0], "duration=")
	for _, secret := range []string{"secret-token", "secret-cursor", "sess-alpha", "sess-beta"} {
		assert.NotContains(t, logs.String(), secret)
	}
}

// newRefreshFailureServer is a fake endpoint that answers initialize. It sets deleted when it gets a session DELETE.
func newRefreshFailureServer(t *testing.T, deleted *atomic.Bool) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted.Store(true)
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "session-1")
		fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"fake","version":"1"}}}`)
	}))
	t.Cleanup(server.Close)
	return server
}

// runRefreshFailureSession serves an initialize and a tools/list whose token expires. Stdin ends with end, or closes
// cleanly when end is nil. It returns what serve printed to stderr, whether the endpoint got a session DELETE, and
// serve's error.
func runRefreshFailureSession(t *testing.T, end error) (string, bool, error) {
	t.Helper()
	var deleted atomic.Bool
	server := newRefreshFailureServer(t, &deleted)
	var stderr bytes.Buffer
	var stdout lockedBuffer
	in := feedAfterAnswers(t, &stdout, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
	}, end)
	cmd := NewMcpServeCmd()
	cmd.SetContext(t.Context())
	cmd.SetErr(&stderr)
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	refresh := &refreshGuard{authenticate: expiresOnToolsList, profile: testProfile, cancel: stop}
	err := serveProxy(ctx, cmd, refresh, proxy.Options{
		In:           in,
		Out:          &stdout,
		Err:          &stderr,
		Endpoint:     mcp.ServiceURL(server.URL, testService),
		Config:       &config.Config{Host: server.URL},
		Authenticate: refresh.authenticateRequest,
		Profile:      testProfile,
	})
	return stderr.String(), deleted.Load(), err
}

// TestServeProxyRefreshFailurePrintsOnceAndExitsTwo checks that a token that expires mid-session fails the request that
// needs it. The bridge logs that failure once, with the profile and the login command. The proxy then exits 2 without
// printing it again, and sends no session DELETE at shutdown.
func TestServeProxyRefreshFailurePrintsOnceAndExitsTwo(t *testing.T) {
	stderr, deleted, err := runRefreshFailureSession(t, nil)

	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
	assert.Equal(t, "databricks aitools mcp serve: upstream: "+refreshRejected.Error()+"\n"+
		"Run: "+refreshLoginHint+"\n"+
		refreshStoppedLine, stderr)
	assert.False(t, deleted, "no session DELETE is sent after a refresh failure")
}

// lockedBuffer collects what serve writes to stdout. The bridge writes from several goroutines.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// feedAfterAnswers returns stdin for serve. It sends each line once the answer to the line before it is on out, as an
// MCP client does: the bridge runs requests concurrently, so a later request could otherwise overtake the initialize.
// The input then ends with end, or closes when end is nil. The answers are the ones the test expects from the fake
// endpoint, so the feeder waits for the ids of the lines it has sent.
func feedAfterAnswers(t *testing.T, out *lockedBuffer, lines []string, end error) io.Reader {
	t.Helper()
	pr, pw := io.Pipe()
	t.Cleanup(func() { pw.Close() })
	go func() {
		for i, line := range lines {
			if i > 0 {
				want := fmt.Sprintf(`"id":%d,`, i)
				deadline := time.Now().Add(10 * time.Second)
				for !strings.Contains(out.String(), want) && time.Now().Before(deadline) {
					time.Sleep(time.Millisecond)
				}
			}
			if _, err := io.WriteString(pw, line+"\n"); err != nil {
				return
			}
		}
		if end != nil {
			pw.CloseWithError(end)
			return
		}
		pw.Close()
	}()
	return pr
}

// TestServeProxyRefreshFailureWinsOverStdinError checks that a stdin error after the refresh failure does not change the
// exit status or the final line.
func TestServeProxyRefreshFailureWinsOverStdinError(t *testing.T) {
	stderr, _, err := runRefreshFailureSession(t, errors.New("input/output error"))

	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
	assert.NotContains(t, stderr, "input/output error")
	assert.True(t, strings.HasSuffix(stderr, refreshStoppedLine), stderr)
}

// signalHelperEnv marks the child process that TestServeSecondSignalEndsProcess starts.
const signalHelperEnv = "AITOOLS_SERVE_SIGNAL_HELPER"

// TestServeSignalHelper is the child of TestServeSecondSignalEndsProcess. It runs the signal context serve uses,
// prints when the first signal cancels it, then stays alive as shutdown would.
func TestServeSignalHelper(t *testing.T) {
	if os.Getenv(signalHelperEnv) == "" {
		t.Skip("runs only as the child of TestServeSecondSignalEndsProcess")
	}
	ctx, cancel := serveSignalContext(t.Context())
	defer cancel()
	fmt.Println("ready")
	<-ctx.Done()
	fmt.Println("cancelled")
	time.Sleep(time.Minute)
}

// TestServeSecondSignalEndsProcess checks that the first signal cancels the context and the second one ends the
// process. A second signal that the handler swallowed during shutdown would leave the child running until the timeout.
func TestServeSecondSignalEndsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Process.Signal cannot send SIGINT on Windows")
	}
	pr, pw, err := os.Pipe()
	require.NoError(t, err)
	defer pr.Close()

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestServeSignalHelper$")
	cmd.Env = append(os.Environ(), signalHelperEnv+"=1")
	cmd.Stdout = pw
	require.NoError(t, cmd.Start())
	require.NoError(t, pw.Close())

	lines := make(chan string)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(pr)
		for sc.Scan() {
			lines <- sc.Text()
		}
	}()
	waitForChildLine(t, lines, "ready")
	require.NoError(t, cmd.Process.Signal(os.Interrupt))
	waitForChildLine(t, lines, "cancelled")
	require.NoError(t, cmd.Process.Signal(os.Interrupt))

	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	select {
	case err := <-exited:
		// A signal ends the child with ExitCode -1. A panic or a normal exit would not, and must fail the test.
		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr, "the child exited without a signal")
		assert.Equal(t, -1, exitErr.ExitCode(), "the child should be ended by the second signal")
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("the second signal did not end the process")
	}
}

// waitForChildLine reads the child's output lines until one equals want. It fails the test when the child closes its
// output first or the wait takes too long.
func waitForChildLine(t *testing.T, lines <-chan string, want string) {
	t.Helper()
	for {
		select {
		case line, ok := <-lines:
			require.True(t, ok, "child exited before it printed %q", want)
			if line == want {
				return
			}
		case <-time.After(20 * time.Second):
			t.Fatalf("timed out waiting for %q", want)
		}
	}
}

// newRedirectingMCPServer is a fake endpoint that does not accept the token for tools/list. That request gets a 303 to a
// sign-in page whose query carries state. Other requests are answered as newEchoMCPServer answers them.
func newRedirectingMCPServer(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusOK)
			return
		}
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if msg.Method == "tools/list" {
			w.Header().Set("Location", "https://sign-in.test/login?state=secret-state")
			w.WriteHeader(http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		result := `{"tools":[]}`
		if msg.Method == "initialize" {
			result = `{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"fake","version":"1"}}`
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, msg.ID, result)
	}))
	t.Cleanup(server.Close)
	return server
}

// TestServeRedirectFailsRequestWithLoginHintAndKeepsServing checks that a 303 from the endpoint fails only the request that
// got it, with the login command for the profile. The Location is not in the output, and serve keeps answering: a later
// request gets the same login error, and stdin closing exits 0.
func TestServeRedirectFailsRequestWithLoginHintAndKeepsServing(t *testing.T) {
	server := newRedirectingMCPServer(t)
	var stdout lockedBuffer
	var stderr bytes.Buffer
	in := feedAfterAnswers(t, &stdout, []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
	}, nil)
	cmd := NewMcpServeCmd()
	cmd.SetContext(t.Context())
	cmd.SetErr(&stderr)

	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	refresh := &refreshGuard{authenticate: func(*http.Request) error { return nil }, profile: "logfood", cancel: stop}
	err := serveProxy(ctx, cmd, refresh, proxy.Options{
		In:        in,
		Out:       &stdout,
		Err:       &stderr,
		Endpoint:  mcp.ServiceURL(server.URL, "system.ai.web_search"),
		Config:    &config.Config{Host: server.URL},
		Profile:   "logfood",
		Transport: &signInRedirect{next: http.DefaultTransport, profile: "logfood"},
	})

	require.NoError(t, err)
	out := stdout.String()
	assert.Contains(t, out, `"id":2,"error"`)
	assert.Contains(t, out, `Run: databricks auth login --profile logfood`)
	assert.Contains(t, out, `"id":3,"error"`)
	assert.NotContains(t, out+stderr.String(), "secret-state")
	assert.NotContains(t, out+stderr.String(), "sign-in.test")
}

// TestSignInRedirectRoundTrip checks that only a 3xx answer becomes the login error. A 200 and a 500 pass through
// unchanged, so the bridge reports them as it does today. A pinned profile's hint names its config file.
func TestSignInRedirectRoundTrip(t *testing.T) {
	const redirectCause = "the MCP endpoint redirected to sign-in (HTTP 303), so it did not accept the token for Databricks profile \"logfood\""
	tests := []struct {
		name       string
		status     int
		configFile string
		wantErr    string
	}{
		{name: "200 passes through", status: http.StatusOK},
		{name: "500 passes through", status: http.StatusInternalServerError},
		{name: "303 fails with the login hint", status: http.StatusSeeOther, wantErr: redirectCause + "\nRun: databricks auth login --profile logfood\nThen restart the MCP server in your agent"},
		{
			name:       "303 with a pinned config file names it",
			status:     http.StatusSeeOther,
			configFile: "/cfg/.databrickscfg",
			wantErr:    redirectCause + "\nRun: DATABRICKS_CONFIG_FILE=/cfg/.databrickscfg databricks auth login --profile logfood\nThen restart the MCP server in your agent",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Location", "https://sign-in.test/login?state=secret-state")
				w.WriteHeader(tc.status)
			}))
			t.Cleanup(server.Close)
			rt := &signInRedirect{next: http.DefaultTransport, profile: "logfood", configFile: tc.configFile}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, server.URL, nil)
			require.NoError(t, err)

			resp, err := rt.RoundTrip(req)

			if tc.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, tc.status, resp.StatusCode)
				resp.Body.Close()
				return
			}
			assert.Nil(t, resp)
			require.EqualError(t, err, tc.wantErr)
			assert.NotContains(t, err.Error(), "secret-state")
		})
	}
}

func TestServeChallengeFollowsStateOrServiceLookup(t *testing.T) {
	const githubPath = "/api/2.1/unity-catalog/mcp-services/system.ai.github"
	const githubJSON = `{"name":"mcp-services/system.ai.github","config":{"source_connection":{"name":"connections/github",` +
		`"options":{"oauth_provider":"github"}}}}`
	var lookups atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != githubPath {
			http.NotFound(w, r)
			return
		}
		lookups.Add(1)
		_, _ = io.WriteString(w, githubJSON)
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	challenge, err := serveChallenge(t.Context(), serveTarget{w: w, entry: &mcp.Entry{SignInRequired: true}, profile: testProfile, service: testService}, io.Discard)
	require.NoError(t, err)
	require.NotNil(t, challenge)
	assert.Equal(t, testService, challenge.Service)
	assert.Equal(t, testProfile, challenge.Profile)
	assert.Equal(t, server.URL, challenge.Host)
	assert.Equal(t, "GitHub", challenge.Label)
	assert.Equal(t, int32(0), lookups.Load(), "an install record answers without a lookup")

	challenge, err = serveChallenge(t.Context(), serveTarget{w: w, entry: &mcp.Entry{SignInRequired: false}, profile: testProfile, service: testService}, io.Discard)
	require.NoError(t, err)
	assert.Nil(t, challenge)
	assert.Equal(t, int32(0), lookups.Load())

	challenge, err = serveChallenge(t.Context(), serveTarget{w: w, profile: testProfile, service: testService}, io.Discard)
	require.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Equal(t, int32(1), lookups.Load(), "without a record the service is read once")
}

// noHostMetadata answers a host-metadata lookup with no metadata, so a client for a .test host does not try to reach it.
func noHostMetadata(context.Context, string) (*config.HostMetadata, error) {
	return nil, nil
}

// TestServeChallengeNamesTheRecordedWorkspace checks that the challenge of a server with an install record names the
// workspace the record holds, which is the workspace the binding check keeps the profile on.
func TestServeChallengeNamesTheRecordedWorkspace(t *testing.T) {
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: "https://workspace.test", Token: "test-token", HostMetadataResolver: noHostMetadata})
	require.NoError(t, err)

	challenge, err := serveChallenge(t.Context(), serveTarget{w: w, entry: &mcp.Entry{SignInRequired: true, WorkspaceID: "42"}, profile: testProfile, service: testService}, io.Discard)
	require.NoError(t, err)
	require.NotNil(t, challenge)
	assert.Equal(t, "42", challenge.WorkspaceID)
}

// TestServeChallengeHintNamesPinnedFileAndQuotesProfile checks that a server pinned to a config file names that file in
// the login command its hints print, and that a profile name with a space is quoted for a shell.
func TestServeChallengeHintNamesPinnedFileAndQuotesProfile(t *testing.T) {
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: "https://workspace.test", Token: "test-token", HostMetadataResolver: noHostMetadata})
	require.NoError(t, err)
	const pinned = "/work/cfg/my file.cfg"

	challenge, err := serveChallenge(t.Context(), serveTarget{w: w, entry: &mcp.Entry{SignInRequired: true, ConfigFile: pinned}, profile: "my profile", service: testService, pinnedFile: pinned}, io.Discard)
	require.NoError(t, err)
	require.NotNil(t, challenge)
	assert.Equal(t, "DATABRICKS_CONFIG_FILE='/work/cfg/my file.cfg' databricks aitools mcp login system.ai.github --profile 'my profile'", challenge.LoginCommand)
	assert.Equal(t, "DATABRICKS_CONFIG_FILE='/work/cfg/my file.cfg' databricks aitools mcp logout system.ai.github --profile 'my profile'", challenge.LogoutCommand)
}

// failingGuard returns a refresh guard whose token cannot be refreshed. cancel stops the serve, as serve's signal
// context does.
func failingGuard(cancel context.CancelFunc) *refreshGuard {
	return &refreshGuard{
		authenticate: func(*http.Request) error { return refreshRejected },
		profile:      testProfile,
		cancel:       cancel,
	}
}

// serviceServer answers the service read of the startup lookup and the credential read of the sign-in with status.
// Its body is the JSON of the answer.
func serviceServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(status)
		_, _ = io.WriteString(rw, body)
	}))
	t.Cleanup(server.Close)
	return server
}

// TestServeChallengeLookupIsBounded checks that the startup read of a service with no install record ends after the
// profile check timeout, as the checks of list --check do, even when the host keeps answering with a retriable error. The
// serve still starts, with the sign-in handling on.
func TestServeChallengeLookupIsBounded(t *testing.T) {
	prev := profileCheckTimeout
	t.Cleanup(func() { profileCheckTimeout = prev })
	profileCheckTimeout = 100 * time.Millisecond
	server := serviceServer(t, http.StatusServiceUnavailable, `{"error_code":"TEMPORARILY_UNAVAILABLE","message":"try later"}`)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	start := time.Now()
	challenge, err := serveChallenge(t.Context(), serveTarget{w: w, profile: testProfile, service: testService}, io.Discard)

	require.NoError(t, err)
	assert.NotNil(t, challenge)
	assert.Less(t, time.Since(start), 10*time.Second, "the lookup must not wait for the SDK's retry budget")
}

// TestServeChallengeLookupFailureStartsWithSignInHandling checks that a startup read which fails for a reason other than
// authentication does not stop the serve. The sign-in handling is on, since it acts only on a challenge that names the
// service, and one line on stderr says that the read failed.
func TestServeChallengeLookupFailureStartsWithSignInHandling(t *testing.T) {
	prev := profileCheckTimeout
	t.Cleanup(func() { profileCheckTimeout = prev })
	profileCheckTimeout = 100 * time.Millisecond
	for name, tc := range map[string]struct {
		status int
		body   string
	}{
		"server error": {http.StatusInternalServerError, `{"error_code":"INTERNAL_ERROR","message":"boom"}`},
		"not found":    {http.StatusNotFound, `{"error_code":"NOT_FOUND","message":"not found"}`},
	} {
		t.Run(name, func(t *testing.T) {
			server := serviceServer(t, tc.status, tc.body)
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
			require.NoError(t, err)
			var stderr bytes.Buffer

			challenge, err := serveChallenge(t.Context(), serveTarget{w: w, profile: testProfile, service: testService}, &stderr)

			require.NoError(t, err)
			require.NotNil(t, challenge)
			assert.Equal(t, testService, challenge.Service)
			assert.Equal(t, testProfile, challenge.Profile)
			lines := strings.Split(strings.TrimSuffix(stderr.String(), "\n"), "\n")
			require.Len(t, lines, 1)
			assert.Contains(t, lines[0], "cannot read MCP service "+testService)
			assert.Contains(t, lines[0], "serving it with sign-in handling on")
		})
	}
}

// TestServeLookupFailureStillOffersAuthenticate checks that a server started after a failed startup read still offers the
// authenticate tool when the service challenges the session, so a failed read does not leave a sign-in-required service
// without its sign-in.
func TestServeLookupFailureStillOffersAuthenticate(t *testing.T) {
	prev := profileCheckTimeout
	t.Cleanup(func() { profileCheckTimeout = prev })
	profileCheckTimeout = 100 * time.Millisecond
	const lookupPath = "/api/2.1/unity-catalog/mcp-services/" + testService
	const endpointPath = "/ai-gateway/mcp-services/" + testService
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case lookupPath:
			rw.WriteHeader(http.StatusServiceUnavailable)
			_, _ = io.WriteString(rw, `{"error_code":"TEMPORARILY_UNAVAILABLE","message":"try later"}`)
		case endpointPath:
			var msg struct {
				ID json.RawMessage `json:"id"`
			}
			body, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(body, &msg)
			rw.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(rw, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32006,"message":"rejected","data":null}}`, msg.ID)
		default:
			rw.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	challenge, err := serveChallenge(t.Context(), serveTarget{w: w, profile: testProfile, service: testService}, io.Discard)
	require.NoError(t, err)
	require.NotNil(t, challenge)

	var out bytes.Buffer
	err = proxy.Run(t.Context(), proxy.Options{
		In:           strings.NewReader(`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n"),
		Out:          &out,
		Err:          io.Discard,
		Endpoint:     mcp.ServiceURL(server.URL, testService),
		Config:       &config.Config{Host: server.URL},
		Authenticate: func(*http.Request) error { return nil },
		Profile:      testProfile,
		Challenge:    challenge,
	})

	require.NoError(t, err)
	assert.Contains(t, out.String(), `"name":"authenticate"`)
}

// TestServeChallengeLookupAuthFailureExitsTwo checks that a startup read that the profile's credential fails names the
// profile and its login command, and exits 2 like the other failures of a credential.
func TestServeChallengeLookupAuthFailureExitsTwo(t *testing.T) {
	server := serviceServer(t, http.StatusUnauthorized, `{"error_code":"UNAUTHENTICATED","message":"Invalid access token."}`)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	_, err = serveChallenge(t.Context(), serveTarget{w: w, profile: testProfile, service: testService}, io.Discard)

	require.Error(t, err)
	assert.True(t, authFailed(err))
	assert.Contains(t, err.Error(), "databricks auth login --profile logfood")
}

// TestServeChallengeLookupHTMLIsAuthFailureExitsTwo checks that a startup read answered with a sign-in page, as the
// workspace answers a credential it rejects, is an auth failure: it names the profile's login command and exits 2.
func TestServeChallengeLookupHTMLIsAuthFailureExitsTwo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(rw, "<html><body>Sign in</body></html>")
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)

	_, err = serveChallenge(t.Context(), serveTarget{w: w, profile: testProfile, service: testService}, io.Discard)

	require.Error(t, err)
	assert.True(t, authFailed(err))
	assert.Contains(t, err.Error(), "databricks auth login --profile logfood")
}

// TestServeChallengeStopsQuietlyOnSignal checks that a signal during the startup read ends the start as a shutdown does:
// the read's cancellation is returned, and nothing is written, rather than a line that says the read failed.
func TestServeChallengeStopsQuietlyOnSignal(t *testing.T) {
	server := serviceServer(t, http.StatusInternalServerError, `{"error_code":"INTERNAL_ERROR","message":"boom"}`)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	var stderr bytes.Buffer

	_, err = serveChallenge(ctx, serveTarget{w: w, profile: testProfile, service: testService}, &stderr)

	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, stderr.String())
}

// TestGuardedClientSendsNoSentinelWorkspaceID checks that a profile whose workspace_id is the CLI's "none" sentinel sends no
// workspace routing header on the requests of the serve's own client, the startup read and the sign-in.
func TestGuardedClientSendsNoSentinelWorkspaceID(t *testing.T) {
	var header http.Header
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		header = r.Header.Clone()
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(rw, `{"error_code":"NOT_FOUND","message":"nope"}`)
	}))
	t.Cleanup(server.Close)
	cfgFile := filepath.Join(t.TempDir(), "cfg")
	body := "[" + testProfile + "]\nhost = " + server.URL + "\ntoken = test-token\nworkspace_id = none\n"
	require.NoError(t, os.WriteFile(cfgFile, []byte(body), 0o600))
	refresh := &refreshGuard{authenticate: func(*http.Request) error { return nil }, profile: testProfile, cancel: func() {}}

	w, err := guardedClient(t.Context(), &config.Config{Profile: testProfile, ConfigFile: cfgFile}, refresh)
	require.NoError(t, err)
	_, err = mcp.GetService(t.Context(), w, testService)

	require.Error(t, err)
	assert.Empty(t, header.Get("X-Databricks-Workspace-Id"))
}

// TestServeRefreshFailureStopsProxy checks that a token that cannot be refreshed during a call the serve makes on its own,
// the startup read or the sign-in, stops the proxy with the login hint, as a refresh failure of a bridge request does.
func TestServeRefreshFailureStopsProxy(t *testing.T) {
	t.Setenv("BROWSER", "none")
	for name, call := range map[string]func(context.Context, *databricks.WorkspaceClient) error{
		"startup read": func(ctx context.Context, w *databricks.WorkspaceClient) error {
			_, err := serveChallenge(ctx, serveTarget{w: w, profile: testProfile, service: testService}, io.Discard)
			return err
		},
		"sign-in": func(ctx context.Context, w *databricks.WorkspaceClient) error {
			signIn := serveSignIn(serveTarget{w: w, profile: testProfile, service: testService})
			_, _, err := signIn(ctx, proxy.SignInRequest{Open: true, Progress: func(string) {}, Stderr: io.Discard})
			return err
		},
	} {
		t.Run(name, func(t *testing.T) {
			server := serviceServer(t, http.StatusBadRequest, `{"error_code":"BAD_REQUEST","message":"bad"}`)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			w := guardedTestClient(t, server.URL, failingGuard(cancel))

			err := call(ctx, w)

			require.Error(t, err)
			assert.Error(t, ctx.Err(), "the proxy is stopped")
			assert.Contains(t, err.Error(), "databricks auth login --profile logfood\nThen restart the MCP server in your agent")
		})
	}
}

// TestServeSignInReportsEachPoll checks that the sign-in of the authenticate tool calls the progress of the request on
// each poll, so the proxy can tell the client that the wait is still going.
func TestServeSignInReportsEachPoll(t *testing.T) {
	prevPoll, prevWait := signInPollInterval, signInWaitTimeout
	t.Cleanup(func() { signInPollInterval, signInWaitTimeout = prevPoll, prevWait })
	signInPollInterval, signInWaitTimeout = time.Millisecond, 10*time.Second
	t.Setenv("BROWSER", "none")

	const credentialPath = "/api/2.1/unity-catalog/mcp-services/system.ai.github/user-credentials"
	var gets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path != credentialPath {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		// The caller signs in on the third read: the first is the check before the sign-in, then two polls.
		if gets.Add(1) < 3 {
			rw.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(rw, `{"error_code":"NOT_FOUND","message":"Credential is not found"}`)
			return
		}
		_, _ = io.WriteString(rw, `{"name":"mcp-services/system.ai.github","provisioning_info":{"state":"ACTIVE"}}`)
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token", WorkspaceID: "1"})
	require.NoError(t, err)

	polls := 0
	signIn := serveSignIn(serveTarget{w: w, profile: testProfile, service: testService})
	_, _, err = signIn(t.Context(), proxy.SignInRequest{Open: true, Progress: func(string) { polls++ }, Stderr: io.Discard})

	require.NoError(t, err)
	assert.Equal(t, 2, polls)
}

// TestServeSignInAuthenticatesEveryRequest checks that each request the sign-in client makes is authenticated by the
// refresh guard, so a token that cannot be refreshed during a sign-in stops the proxy, whichever call hits it.
func TestServeSignInAuthenticatesEveryRequest(t *testing.T) {
	prevPoll, prevWait := signInPollInterval, signInWaitTimeout
	t.Cleanup(func() { signInPollInterval, signInWaitTimeout = prevPoll, prevWait })
	signInPollInterval, signInWaitTimeout = time.Millisecond, 10*time.Second
	t.Setenv("BROWSER", "none")

	const credentialPath = "/api/2.1/unity-catalog/mcp-services/system.ai.github/user-credentials"
	var reads atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.URL.Path != credentialPath {
			rw.WriteHeader(http.StatusNotFound)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		if reads.Add(1) < 3 {
			rw.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(rw, `{"error_code":"NOT_FOUND","message":"Credential is not found"}`)
			return
		}
		_, _ = io.WriteString(rw, `{"name":"mcp-services/system.ai.github","provisioning_info":{"state":"ACTIVE"}}`)
	}))
	t.Cleanup(server.Close)
	var authenticated atomic.Int32
	guard := &refreshGuard{authenticate: func(*http.Request) error { authenticated.Add(1); return nil }, profile: testProfile, cancel: func() {}}
	w := guardedTestClient(t, server.URL, guard)

	signIn := serveSignIn(serveTarget{w: w, profile: testProfile, service: testService})
	_, _, err := signIn(t.Context(), proxy.SignInRequest{Open: true, Progress: func(string) {}, Stderr: io.Discard})

	require.NoError(t, err)
	assert.Equal(t, reads.Load(), authenticated.Load(), "every request of the sign-in is authenticated by the guard")
}

// TestServeSignInUsesTheRecordedWorkspaceID checks that a sign-in for a recorded server takes the workspace ID from the
// record, so it makes no request for it.
func TestServeSignInUsesTheRecordedWorkspaceID(t *testing.T) {
	t.Setenv("BROWSER", "none")
	var paths []string
	var mu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		_, _ = io.WriteString(rw, `{"name":"mcp-services/system.ai.github","provisioning_info":{"state":"ACTIVE"}}`)
	}))
	t.Cleanup(server.Close)
	guard := &refreshGuard{authenticate: func(*http.Request) error { return nil }, profile: testProfile, cancel: func() {}}
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Credentials: guardCredentials{refresh: guard}})
	require.NoError(t, err)

	var pages []string
	signIn := serveSignIn(serveTarget{w: w, entry: &mcp.Entry{WorkspaceID: "42"}, profile: testProfile, service: testService})
	_, _, err = signIn(t.Context(), proxy.SignInRequest{Open: true, Progress: func(page string) { pages = append(pages, page) }, Stderr: io.Discard})

	require.NoError(t, err)
	mu.Lock()
	defer mu.Unlock()
	for _, path := range paths {
		assert.NotContains(t, path, "workspace", "no request reads the workspace ID")
	}
	for _, page := range pages {
		assert.Contains(t, page, "o=42")
	}
}

// TestServeSignInTransientFailureKeepsServing checks that a token failure which is not a rejection fails only the
// sign-in's request, with the transient message, and does not stop the proxy.
func TestServeSignInTransientFailureKeepsServing(t *testing.T) {
	t.Setenv("BROWSER", "none")
	server := serviceServer(t, http.StatusOK, `{}`)
	var stopped bool
	guard := &refreshGuard{
		authenticate: func(*http.Request) error { return errors.New("dial tcp: i/o timeout") },
		profile:      testProfile,
		cancel:       func() { stopped = true },
	}
	w := guardedTestClient(t, server.URL, guard)
	signIn := serveSignIn(serveTarget{w: w, profile: testProfile, service: testService})

	_, _, err := signIn(t.Context(), proxy.SignInRequest{Open: true, Progress: func(string) {}, Stderr: io.Discard})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "the next request tries again")
	assert.NotContains(t, err.Error(), "restart the MCP server")
	assert.False(t, stopped, "the proxy keeps running")
	assert.NoError(t, guard.failure())
}

// guardedTestClient returns a client for host whose requests the guard authenticates, as the sign-in client of serve is.
func guardedTestClient(t *testing.T, host string, guard *refreshGuard) *databricks.WorkspaceClient {
	t.Helper()
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: host, Credentials: guardCredentials{refresh: guard}, WorkspaceID: "1"})
	require.NoError(t, err)
	return w
}

// TestServeSignInProgressNamesThePage checks that the progress of a sign-in names its page on every poll, whether or not
// this attempt opens the page. A client that hides stderr sees the page only through progress.
func TestServeSignInProgressNamesThePage(t *testing.T) {
	prevPoll, prevWait := signInPollInterval, signInWaitTimeout
	t.Cleanup(func() { signInPollInterval, signInWaitTimeout = prevPoll, prevWait })
	signInPollInterval, signInWaitTimeout = time.Millisecond, 10*time.Second
	t.Setenv("BROWSER", "none")

	const credentialPath = "/api/2.1/unity-catalog/mcp-services/system.ai.github/user-credentials"
	for _, tc := range []struct {
		name string
		open bool
	}{
		{name: "page opened", open: true},
		{name: "page not opened", open: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reads atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				if r.URL.Path != credentialPath {
					rw.WriteHeader(http.StatusNotFound)
					return
				}
				rw.Header().Set("Content-Type", "application/json")
				if reads.Add(1) < 3 {
					rw.WriteHeader(http.StatusNotFound)
					_, _ = io.WriteString(rw, `{"error_code":"NOT_FOUND","message":"Credential is not found"}`)
					return
				}
				_, _ = io.WriteString(rw, `{"name":"mcp-services/system.ai.github","provisioning_info":{"state":"ACTIVE"}}`)
			}))
			t.Cleanup(server.Close)
			w := guardedTestClient(t, server.URL, &refreshGuard{authenticate: func(*http.Request) error { return nil }, profile: testProfile, cancel: func() {}})
			var pageURLs []string
			signIn := serveSignIn(serveTarget{w: w, profile: testProfile, service: testService})

			_, _, err := signIn(t.Context(), proxy.SignInRequest{Open: tc.open, Stderr: io.Discard, Progress: func(pageURL string) {
				pageURLs = append(pageURLs, pageURL)
			}})

			require.NoError(t, err)
			// The fixture is read twice before the caller signs in: the check before the sign-in, then two polls.
			want, err := mcp.SignInURL(server.URL, testService, "1")
			require.NoError(t, err)
			assert.Equal(t, []string{want, want}, pageURLs)
		})
	}
}

// TestServeSignInPageIsNotCancelledWithTheRequest checks that the page of a sign-in whose request is cancelled is opened with
// a context that the cancellation does not reach. A browser command the request's context started would be killed with the
// request, and the page the user is on with it.
func TestServeSignInPageIsNotCancelledWithTheRequest(t *testing.T) {
	prevPoll, prevWait := signInPollInterval, signInWaitTimeout
	t.Cleanup(func() { signInPollInterval, signInWaitTimeout = prevPoll, prevWait })
	signInPollInterval, signInWaitTimeout = time.Millisecond, 10*time.Second
	t.Setenv("BROWSER", "")
	prevOpen := serveOpenPage
	t.Cleanup(func() { serveOpenPage = prevOpen })
	entered, release := make(chan struct{}), make(chan struct{})
	pageErr := make(chan error, 1)
	serveOpenPage = func(ctx context.Context, _ string) error {
		close(entered)
		<-release
		pageErr <- ctx.Err()
		return nil
	}

	// The credential is never signed in, so the sign-in waits until its request is cancelled.
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(rw, `{"error_code":"NOT_FOUND","message":"Credential is not found"}`)
	}))
	t.Cleanup(server.Close)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token", WorkspaceID: "1"})
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	signInErr := make(chan error, 1)
	signIn := serveSignIn(serveTarget{w: w, profile: testProfile, service: testService})
	go func() {
		_, _, err := signIn(ctx, proxy.SignInRequest{Open: true, Progress: func(string) {}, Stderr: io.Discard})
		signInErr <- err
	}()
	select {
	case <-entered:
	case <-time.After(10 * time.Second):
		t.Fatal("the page did not open")
	}
	cancel()
	close(release)

	require.ErrorIs(t, <-signInErr, context.Canceled)
	assert.NoError(t, <-pageErr, "the page is opened with a context the request's cancellation does not reach")
}

// TestGuardedClientNotAnAgentDropsAgentHeader checks that the serve's client leaves the agent token out of User-Agent
// under DATABRICKS_NOT_AN_AGENT, and keeps it without the variable.
func TestGuardedClientNotAnAgentDropsAgentHeader(t *testing.T) {
	t.Setenv("CLAUDECODE", "1")
	useragent.ClearCache()
	t.Cleanup(useragent.ClearCache)

	cases := []struct {
		name    string
		noAgent bool
	}{
		{name: "variable set", noAgent: true},
		{name: "variable unset", noAgent: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var userAgent string
			server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
				userAgent = r.Header.Get("User-Agent")
				rw.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(rw, `{"error_code":"NOT_FOUND","message":"nope"}`)
			}))
			t.Cleanup(server.Close)
			cfgFile := filepath.Join(t.TempDir(), "cfg")
			body := "[" + testProfile + "]\nhost = " + server.URL + "\ntoken = test-token\nworkspace_id = none\n"
			require.NoError(t, os.WriteFile(cfgFile, []byte(body), 0o600))
			refresh := &refreshGuard{authenticate: func(*http.Request) error { return nil }, profile: testProfile, cancel: func() {}}

			ctx := t.Context()
			if tc.noAgent {
				t.Setenv("DATABRICKS_NOT_AN_AGENT", "1")
			}
			w, err := guardedClient(ctx, &config.Config{Profile: testProfile, ConfigFile: cfgFile}, refresh)
			require.NoError(t, err)
			// The service call fails on the 404; only the User-Agent of the request matters here.
			_, _ = mcp.GetService(ctx, w, testService)

			require.NotEmpty(t, userAgent)
			if tc.noAgent {
				assert.NotContains(t, userAgent, "agent/")
			} else {
				assert.Contains(t, userAgent, "agent/claude-code")
			}
		})
	}
}
