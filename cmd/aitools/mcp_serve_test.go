package aitools

import (
	"bufio"
	"bytes"
	"context"
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
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	argv := []string{mcpconfig.ProfileFlag, "logfood", mcpconfig.ServiceFlag, "system.ai.github", mcpconfig.AgentFlag, "claude-code"}
	require.NoError(t, serve.ParseFlags(argv))

	profile, err := serve.Flags().GetString("profile")
	require.NoError(t, err)
	service, err := serve.Flags().GetString("service")
	require.NoError(t, err)
	agent, err := serve.Flags().GetString("agent")
	require.NoError(t, err)
	assert.Equal(t, []string{"logfood", "system.ai.github", "claude-code"}, []string{profile, service, agent})
}

func TestRefreshGuardPassesThroughWhileTokensRefresh(t *testing.T) {
	cancelled := false
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return nil },
		profile:      "logfood",
		cancel:       func() { cancelled = true },
	}

	require.NoError(t, g.authenticateRequest(&http.Request{Header: http.Header{}}))
	assert.NoError(t, g.failure())
	assert.False(t, cancelled)
}

func TestRefreshGuardStopsProxyOnFirstFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	calls := 0
	cause := errors.New(refreshTokenExpired)
	g := &refreshGuard{
		authenticate: func(*http.Request) error {
			calls++
			return cause
		},
		profile: "logfood",
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
	cause := errors.New(refreshTokenExpired)
	g := &refreshGuard{
		authenticate: func(*http.Request) error {
			calls++
			return cause
		},
		profile: "logfood",
		cancel:  func() {},
	}
	require.Error(t, g.authenticateRequest(&http.Request{Header: http.Header{}}))

	err := g.authenticateRequest(&http.Request{Header: http.Header{}})

	assert.Equal(t, 1, calls)
	assert.EqualError(t, err, "the session is stopping after a token refresh failure for profile logfood")
	assert.ErrorIs(t, g.failure(), cause)
}

// TestRefreshGuardStoresAuthError checks that the refresh failure reaches the user as the AUTH_FAILED error, which names
// the profile and its login command.
func TestRefreshGuardStoresAuthError(t *testing.T) {
	cause := errors.New(refreshTokenExpired)
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

func TestServeFailurePrintsOnceWithStatus(t *testing.T) {
	cmd := NewMcpServeCmd()
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	cause := mcp.AuthError("logfood", errors.New(refreshTokenExpired))

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
		profile:      "logfood",
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
		authenticate: func(*http.Request) error { return errors.New(refreshTokenExpired) },
		profile:      "logfood",
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
	for _, service := range []string{"github", "system.ai", "system..github", "a.b.c.d"} {
		t.Run(service, func(t *testing.T) {
			parent := newMCPTestRoot()
			parent.SetArgs([]string{"mcp", "serve", "--profile", "logfood", "--service", service})
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
	parent.SetArgs([]string{"mcp", "serve", "--profile", "logfood", "--service", "system.ai.github"})
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
			return errors.New(refreshTokenExpired)
		},
		profile: "logfood",
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
	cause := errors.New(refreshTokenExpired)
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return cause },
		profile:      "logfood",
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
	g := &refreshGuard{profile: "logfood"}
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

// expiresOnToolsList is an Authenticate whose token expires with the tools/list request, as a refresh token can
// mid-session. It reads the request body, so it answers POST requests only.
func expiresOnToolsList(r *http.Request) error {
	if r.Method != http.MethodPost {
		return nil
	}
	body, err := r.GetBody()
	if err != nil {
		return err
	}
	data, err := io.ReadAll(body)
	body.Close()
	if err != nil {
		return err
	}
	if strings.Contains(string(data), `"tools/list"`) {
		return errors.New(refreshTokenExpired)
	}
	return nil
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
	err := serveProxy(cmd, proxy.Options{
		In:           in,
		Out:          &stdout,
		Err:          &stderr,
		Endpoint:     mcp.ServiceURL(server.URL, "system.ai.github"),
		Config:       &config.Config{Host: server.URL},
		Authenticate: expiresOnToolsList,
		Profile:      "logfood",
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
	assert.Equal(t, "databricks aitools mcp serve: upstream: "+refreshTokenExpired+"\n"+
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
