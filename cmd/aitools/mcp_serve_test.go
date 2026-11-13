package aitools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
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
	assert.EqualError(t, err, "the session is stopping after a token refresh failure for profile logfood")
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

// TestRefreshGuardTransientFailureFailsOnlyThatRequest checks that a token failure which is not a rejection fails that
// request with the login hint and nothing else. The proxy keeps running, and the next request asks for a token again.
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

// TestCredentialRejected checks which token failures stop the proxy. A rejection is one that a new login fixes. Every
// other failure is transient, including the ones this check cannot classify.
func TestCredentialRejected(t *testing.T) {
	oauthResponse := func(status int, body string) error {
		return &httpclient.HttpError{Response: &http.Response{StatusCode: status}, Message: body}
	}
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
	cause := mcp.AuthError(testProfile, "", errors.New(refreshTokenExpired))

	err := serveFailure(cmd, cause, serveBindStatus)

	assert.Equal(t, cause.Error()+"\n", stderr.String())
	assert.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
	assert.ErrorIs(t, err, cause)
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
	for _, service := range []string{"github", "system.ai", "system..github", "a.b.c.d", "system.ai.git#hub", "system.ai.git?hub", "system.ai.git/hub"} {
		t.Run(service, func(t *testing.T) {
			parent := newMCPTestRoot()
			parent.SetArgs([]string{"mcp", "serve", "--profile", testProfile, "--service", service})
			err := parent.ExecuteContext(cmdio.MockDiscard(t.Context()))
			require.ErrorContains(t, err, "--service needs a full service name such as system.ai.github")
			assert.Equal(t, 1, root.ExitStatus(err))
		})
	}
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
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	err := proxyResult(cmd, g, errors.New("read stdin: input/output error"))

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
	assert.Equal(t, "read stdin: input/output error\n", stderr.String())
}

// The refresh failure fixtures: the cause that the fake token gives, the login hint that serve prints with it, and the
// final line that serve prints once the proxy has stopped.
const (
	refreshTokenExpired = "refresh token expired"
	refreshLoginHint    = "databricks auth login --profile logfood, then restart the MCP server in your agent"
	refreshStoppedLine  = "stopped after a token refresh failure for profile logfood\n"
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
		w.Header().Set("Mcp-Session-Id", "session-1")
		result := `{"tools":[]}`
		if msg.Method == "initialize" {
			result = `{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"fake","version":"1"}}`
		}
		fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, msg.ID, result)
	}))
	t.Cleanup(server.Close)
	return server
}

// TestServeProxyTransientRefreshFailureKeepsServing checks that a token failure which is not a rejection fails only the
// request that needed the token. The agent gets the error with the login hint. Serve keeps running, the next request
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

// TestServeChallengeHintNamesPinnedFileAndQuotesProfile checks that a server pinned to a config file names that file in
// the login command its hints print, and that a profile name with a space is quoted for a shell.
func TestServeChallengeHintNamesPinnedFileAndQuotesProfile(t *testing.T) {
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: "https://workspace.test", Token: "test-token"})
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
	assert.Equal(t, serveBindStatus, serveStartStatus(err))
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

// TestServeStartStatusIsOneForOtherFailures checks that a startup failure that is not a credential's exits 1.
func TestServeStartStatusIsOneForOtherFailures(t *testing.T) {
	assert.Equal(t, 1, serveStartStatus(errors.New("cannot read MCP service system.ai.github: not found")))
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
			assert.Equal(t, serveBindStatus, serveStartStatus(err))
			assert.Contains(t, err.Error(), "databricks auth login --profile logfood, then restart the MCP server in your agent")
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

	_, _, err := signIn(t.Context(), proxy.SignInRequest{Open: true, Stderr: io.Discard})

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
			want := mcp.SignInURL(server.URL, testService, "1")
			assert.Equal(t, []string{want, want}, pageURLs)
		})
	}
}
