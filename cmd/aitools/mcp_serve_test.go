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
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/aitools/mcp"
	"github.com/databricks/cli/libs/aitools/mcp/proxy"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMcpServeIsHidden(t *testing.T) {
	assert.True(t, NewMcpServeCmd().Hidden)
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
	cause := errors.New("refresh token expired")
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
	assert.Contains(t, err.Error(), "databricks auth login --profile logfood")
	assert.Equal(t, context.Canceled, ctx.Err())

	// A second failure keeps the first error for the exit status.
	_ = g.authenticateRequest(&http.Request{Header: http.Header{}})
	assert.Equal(t, 2, calls)
	assert.ErrorIs(t, g.failure(), cause)
}

// The refresh failure reaches the user as the AUTH_FAILED error, which names the profile and its login command
// and exits 2.
func TestRefreshGuardStoresAuthError(t *testing.T) {
	cause := errors.New("refresh token expired")
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
	cause := mcp.AuthError("logfood", errors.New("refresh token expired"))

	err := serveFailure(cmd, cause, serveBindStatus)

	assert.Equal(t, cause.Error()+"\n", stderr.String())
	assert.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
	assert.ErrorIs(t, err, cause)
}

func TestServeDialerHasConnectTimeout(t *testing.T) {
	assert.Equal(t, 10*time.Second, serveDialer().Timeout)
}

// The profile's insecure_skip_verify reaches the transport, and HTTP/2 is kept with it.
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

// A request whose own context ended (the client cancelled it, or shutdown began) fails that request only: the
// token was not shown to be unusable.
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

// The session DELETE at shutdown is best effort. Its auth failure is logged by the bridge, and a normal shutdown
// exits 0.
func TestRefreshGuardIgnoresSessionDelete(t *testing.T) {
	cancelled := false
	g := &refreshGuard{
		authenticate: func(*http.Request) error { return errors.New("refresh token expired") },
		profile:      "logfood",
		cancel:       func() { cancelled = true },
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "https://workspace.test/mcp", nil)
	require.NoError(t, err)

	require.Error(t, g.authenticateRequest(req))

	assert.NoError(t, g.failure())
	assert.False(t, cancelled)
}

// An unknown agent is left out of the User-Agent and reported once on stderr, so the server still serves.
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

// A service that is not a full three-part name is refused before any binding check, with the flag named.
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

// A state file that cannot be read stops the binding checks, so it exits 2 like the other binding failures.
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

// After a refresh failure the session DELETE is sent without a credential: the session is over, and a failed DELETE
// would log the refresh error a second time. Authenticate is not asked.
func TestRefreshGuardSendsDeleteWithoutCredentialAfterFailure(t *testing.T) {
	asked := 0
	g := &refreshGuard{
		authenticate: func(*http.Request) error {
			asked++
			return errors.New("refresh token expired")
		},
		profile: "logfood",
		cancel:  func() {},
	}
	require.Error(t, g.authenticateRequest(&http.Request{Header: http.Header{}}))
	asked = 0

	req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "https://workspace.test/mcp", nil)
	require.NoError(t, err)
	assert.NoError(t, g.authenticateRequest(req))
	assert.Equal(t, 0, asked)
}

// A token that expires mid-session fails the request that needs it. The bridge logs that failure once, with the
// profile and the login command; the proxy then exits 2 without printing it again, and the session DELETE at
// shutdown is still sent.
func TestServeProxyRefreshFailurePrintsOnceAndExitsTwo(t *testing.T) {
	var deleted atomic.Bool
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
	defer server.Close()

	// The token expires with the tools/list request. From then on the session DELETE fails too, as an expired
	// token would.
	const refreshErr = "refresh token expired"
	var expired atomic.Bool
	authenticate := func(r *http.Request) error {
		if r.Method != http.MethodPost {
			if expired.Load() {
				return errors.New(refreshErr)
			}
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
			expired.Store(true)
			return errors.New(refreshErr)
		}
		return nil
	}

	stdin := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n" +
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n"
	var stderr bytes.Buffer
	cmd := NewMcpServeCmd()
	cmd.SetContext(t.Context())
	err := serveProxy(cmd, proxy.Options{
		In:           strings.NewReader(stdin),
		Out:          io.Discard,
		Err:          &stderr,
		Endpoint:     mcp.ServiceURL(server.URL, "system.ai.github"),
		Config:       &config.Config{Host: server.URL},
		Authenticate: authenticate,
		Profile:      "logfood",
	})

	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Equal(t, 2, root.ExitStatus(err))
	assert.Equal(t, "databricks aitools mcp serve: upstream: refresh token expired\nRun: databricks auth login --profile logfood\n", stderr.String())
	assert.True(t, deleted.Load(), "the session DELETE is still sent")
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

// The first signal cancels the context and the second one ends the process. A second signal that the handler
// swallowed during shutdown would leave the child running until the timeout.
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
