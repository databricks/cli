package sandbox_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/libs/env"
	"github.com/gorilla/websocket"

	"github.com/databricks/cli/experimental/ssh/sandbox/internal/sandboxexec"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitForRunningLifecycle(t *testing.T) {
	for _, states := range [][]string{
		{"RUNNING"}, {"STOPPED", "RUNNING"}, {"STOPPING", "STOPPED", "PENDING", "RUNNING"}, {"PENDING", "RUNNING"}, {"UNSPECIFIED"}, {"STOPPED", "STOPPED", "RUNNING"}, {"STOPPED", "PENDING", "STOPPED"},
	} {
		t.Run(strings.Join(states, "-"), func(t *testing.T) {
			index, starts := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/.well-known/databricks-config" {
					_, _ = io.WriteString(w, `{}`)
					return
				}
				assert.Equal(t, "/api/2.0/sandboxes/test", strings.TrimSuffix(r.URL.Path, "/start"))
				assert.Equal(t, "123", r.Header.Get("X-Databricks-Workspace-Id"))
				if r.Method == http.MethodPost {
					starts++
					_, _ = io.WriteString(w, `{}`)
					return
				}
				state := states[min(index, len(states)-1)]
				index++
				_ = json.NewEncoder(w).Encode(map[string]any{"status": map[string]string{"state": "SANDBOX_STATE_" + state}})
			}))
			defer server.Close()
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token", WorkspaceID: "123"})
			require.NoError(t, err)
			err = sandbox.WaitForRunning(t.Context(), w, "sandboxes/test")
			if states[len(states)-1] != "RUNNING" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			if strings.Contains(strings.Join(states, "-"), "STOPPED") {
				assert.Equal(t, 1, starts)
			} else {
				assert.Zero(t, starts)
			}
		})
	}
}

type responseStream struct {
	events []*sandboxexec.ExecuteCommandResponse
}

func (stream *responseStream) Recv() (*sandboxexec.ExecuteCommandResponse, error) {
	if len(stream.events) == 0 {
		return nil, io.EOF
	}
	event := stream.events[0]
	stream.events = stream.events[1:]
	return event, nil
}

func TestReadServerOutput(t *testing.T) {
	response := func(data string) *sandboxexec.ExecuteCommandResponse {
		return &sandboxexec.ExecuteCommandResponse{ProcessEvent: &sandboxexec.ProcessEvent{Event: &sandboxexec.ProcessEvent_Data{Data: &sandboxexec.DataEvent{Output: &sandboxexec.DataEvent_Stdout{Stdout: []byte(data)}}}}}
	}
	for _, tc := range []struct {
		name      string
		chunks    []string
		port      int
		errorText string
	}{
		{"fragmented", []string{"{\"port\":4", "0000}\n"}, 40000, "EOF"},
		{"invalid-json", []string{"hello\n"}, 0, "invalid character"},
		{"invalid-port", []string{"{\"port\":22}\n"}, 0, "invalid sandbox relay port"},
		{"too-large", []string{strings.Repeat("x", 4097)}, 0, "too large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stream := &responseStream{}
			for _, chunk := range tc.chunks {
				stream.events = append(stream.events, response(chunk))
			}
			ready := make(chan sandbox.Readiness, 1)
			stopped := make(chan error, 1)
			sandbox.ReadServerOutputForTest(stream, "command", ready, stopped, io.Discard)
			require.ErrorContains(t, <-stopped, tc.errorText)
			if tc.port != 0 {
				assert.Equal(t, tc.port, (<-ready).Port)
			} else {
				assert.Empty(t, ready)
			}
		})
	}
	ready := make(chan sandbox.Readiness, 1)
	stopped := make(chan error, 1)
	stream := &responseStream{events: []*sandboxexec.ExecuteCommandResponse{{ProcessEvent: &sandboxexec.ProcessEvent{Event: &sandboxexec.ProcessEvent_Start{Start: &sandboxexec.StartEvent{CommandId: new("wrong")}}}}}}
	sandbox.ReadServerOutputForTest(stream, "expected", ready, stopped, io.Discard)
	require.ErrorContains(t, <-stopped, "unexpected command ID")
}

func TestForwardingURL(t *testing.T) {
	endpoint, err := sandbox.ForwardingURLForTest("https://workspace.test", "sandboxes/test", 40000, "connection")
	require.NoError(t, err)
	assert.Equal(t, "wss://workspace.test/sandbox-port-forward/sandboxes/test/ports/40000/ssh?id=connection", endpoint)
	_, err = sandbox.ForwardingURLForTest("http://workspace.test", "sandboxes/test", 40000, "connection")
	require.Error(t, err)
	_, err = sandbox.ForwardingURLForTest("https://workspace.test", "sandboxes/../other", 40000, "connection")
	require.Error(t, err)
}

func TestPrepareRemoteRejectsPartialSetup(t *testing.T) {
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	identity, err := sandbox.LocalIdentity(ctx, "https://workspace.test", "caller", "sandboxes/test")
	require.NoError(t, err)
	valid, _ := json.Marshal(map[string]string{"home": "/home/sandbox-agent", "config": "/home/sandbox-agent/config", "sshd": "/usr/sbin/sshd", "user": "sandbox-agent", "host_key": string(identity.PublicKey), "session_dir": "/home/sandbox-agent/session", "architecture": "x86_64"})
	for _, tc := range []struct {
		name, status, stdout string
		code                 int
		truncated            bool
		errorText            string
	}{
		{name: "valid", status: "COMPLETED", stdout: string(valid)},
		{name: "failed", status: "FAILED", errorText: "setup failed"},
		{name: "exit", status: "COMPLETED", code: 2, errorText: "setup failed"},
		{name: "truncated", status: "COMPLETED", truncated: true, errorText: "setup failed"},
		{name: "invalid-json", status: "COMPLETED", stdout: "hello", errorText: "read sandbox SSH configuration"},
		{name: "unsupported-user", status: "COMPLETED", stdout: `{"user":"root"}`, errorText: "requires the sandbox-agent account"},
		{name: "incomplete", status: "COMPLETED", stdout: `{"user":"sandbox-agent"}`, errorText: "incomplete sandbox SSH configuration"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/.well-known/databricks-config" {
					_, _ = io.WriteString(w, `{}`)
					return
				}
				assert.Equal(t, "/api/2.0/sandbox-exec/sandboxes/test/exec-sync", r.URL.Path)
				_ = json.NewEncoder(w).Encode(map[string]any{"status": "EXECUTE_COMMAND_STATUS_" + tc.status, "stdout": tc.stdout, "exit_code": tc.code, "truncated": tc.truncated})
			}))
			defer server.Close()
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
			require.NoError(t, err)
			_, err = sandbox.PrepareRemoteForTest(ctx, w, "sandboxes/test", identity)
			if tc.errorText != "" {
				require.ErrorContains(t, err, tc.errorText)
			} else {
				require.NoError(t, err)
				contents, err := os.ReadFile(identity.KnownHostsPath)
				require.NoError(t, err)
				assert.Contains(t, string(contents), identity.HostKeyAlias)
			}
		})
	}
}

// Closing an inherited blocking descriptor need not interrupt an in-flight read.
type blockedInput struct{ started, release, finished chan struct{} }

func (input *blockedInput) Read([]byte) (int, error) {
	close(input.started)
	<-input.release
	close(input.finished)
	return 0, io.EOF
}
func (*blockedInput) Close() error { return nil }

func TestProxyStreamStopsWithoutWaitingForBlockedStdin(t *testing.T) {
	input := &blockedInput{started: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
	defer func() { close(input.release); <-input.finished }()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteMessage(websocket.BinaryMessage, []byte("SSH-2.0-test\r\n"))
		<-input.started
		_ = conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
	}))
	defer server.Close()
	finished := make(chan error, 1)
	go func() {
		finished <- sandbox.ProxyStreamForTest(t.Context(), input, io.Discard, func(ctx context.Context, _ proxy.DialRequest) (*websocket.Conn, error) {
			conn, response, err := websocket.DefaultDialer.DialContext(ctx, strings.Replace(server.URL, "http://", "ws://", 1), nil)
			if response != nil {
				response.Body.Close()
			}
			return conn, err
		})
	}()
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("proxy waited for blocked inherited stdin after WebSocket closure")
	}
}
