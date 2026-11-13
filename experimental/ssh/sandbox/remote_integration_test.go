package sandbox_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/experimental/ssh/sandbox/internal/sandboxexec"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

type execService any

type remoteFixture struct {
	mode                          string
	publicKey                     string
	connected                     chan struct{}
	mu                            sync.Mutex
	terminated, removed, uploaded bool
	authenticated                 bool
	commandID                     string
}

func (fixture *remoteFixture) execute(_ any, stream grpc.ServerStream) error {
	request := &sandboxexec.ExecuteCommandRequest{}
	if err := stream.RecvMsg(request); err != nil {
		return err
	}
	fixture.mu.Lock()
	fixture.commandID = request.GetCommandId()
	fixture.mu.Unlock()
	if err := stream.SendMsg(&sandboxexec.ExecuteCommandResponse{ProcessEvent: &sandboxexec.ProcessEvent{Event: &sandboxexec.ProcessEvent_Start{Start: &sandboxexec.StartEvent{CommandId: request.CommandId}}}}); err != nil {
		return err
	}
	switch fixture.mode {
	case "before-ready":
	case "cancel", "startup-timeout":
		<-stream.Context().Done()
		return stream.Context().Err()
	case "after-ready", "clean-close", "slow-session":
		if err := stream.SendMsg(&sandboxexec.ExecuteCommandResponse{ProcessEvent: &sandboxexec.ProcessEvent{Event: &sandboxexec.ProcessEvent_Data{Data: &sandboxexec.DataEvent{Output: &sandboxexec.DataEvent_Stdout{Stdout: []byte("{\"port\":40000}\n")}}}}}); err != nil {
			return err
		}
		select {
		case <-fixture.connected:
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
		if fixture.mode == "clean-close" || fixture.mode == "slow-session" {
			<-stream.Context().Done()
			return stream.Context().Err()
		}
	}
	return stream.SendMsg(&sandboxexec.ExecuteCommandResponse{ProcessEvent: &sandboxexec.ProcessEvent{Event: &sandboxexec.ProcessEvent_End{End: &sandboxexec.EndEvent{ExitCode: proto.Int32(17)}}}})
}

func (fixture *remoteFixture) terminate(_ any, _ context.Context, decode func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
	request := &sandboxexec.TerminateCommandRequest{}
	if err := decode(request); err != nil {
		return nil, err
	}
	fixture.mu.Lock()
	defer fixture.mu.Unlock()
	if request.GetCommandId() != fixture.commandID {
		return nil, errors.New("termination used a different command ID")
	}
	fixture.terminated = true
	return &sandboxexec.TerminateCommandResponse{}, nil
}

func (fixture *remoteFixture) serveHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/.well-known/databricks-config":
		_, _ = io.WriteString(w, `{}`)
	case r.URL.Path == "/api/2.0/sandboxes/test":
		_, _ = io.WriteString(w, `{"status":{"state":"SANDBOX_STATE_RUNNING"}}`)
	case strings.HasPrefix(r.URL.Path, "/api/2.0/sandbox-files/"):
		fixture.mu.Lock()
		fixture.uploaded = true
		fixture.mu.Unlock()
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusCreated)
	case strings.HasSuffix(r.URL.Path, "/exec-sync"):
		var request struct {
			Cmd  string   `json:"cmd"`
			Args []string `json:"args"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		result := map[string]any{"status": "EXECUTE_COMMAND_STATUS_COMPLETED"}
		switch request.Cmd {
		case "python3":
			setup, _ := json.Marshal(map[string]string{"home": "/home/sandbox-agent", "config": "/home/sandbox-agent/session/sshd_config", "sshd": "/usr/sbin/sshd", "user": "sandbox-agent", "host_key": fixture.publicKey, "session_dir": "/home/sandbox-agent/session", "architecture": "x86_64"})
			result["stdout"] = string(setup)
		case "rm":
			fixture.mu.Lock()
			fixture.removed = true
			fixture.mu.Unlock()
		}
		_ = json.NewEncoder(w).Encode(result)
	case strings.HasPrefix(r.URL.Path, "/sandbox-port-forward/"):
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-Databricks-Workspace-Id") != "123" {
			http.Error(w, "missing forwarding credentials", http.StatusUnauthorized)
			return
		}
		fixture.mu.Lock()
		fixture.authenticated = true
		fixture.mu.Unlock()
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteMessage(websocket.BinaryMessage, []byte("SSH-2.0-test\r\n"))
		close(fixture.connected)
		if fixture.mode == "slow-session" {
			time.Sleep(1500 * time.Millisecond)
		}
		if fixture.mode == "clean-close" || fixture.mode == "slow-session" {
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	default:
		http.NotFound(w, r)
	}
}

func TestConnectProxyCleansUpAfterCompletionFailureAndCancellation(t *testing.T) {
	for _, mode := range []string{"before-ready", "after-ready", "cancel", "clean-close", "startup-timeout", "slow-session"} {
		t.Run(mode, func(t *testing.T) {
			home := t.TempDir()
			ctx := env.WithUserHomeDir(t.Context(), home)
			identity, err := sandbox.LocalIdentity(ctx, "https://workspace.test", "caller", "sandboxes/test")
			require.NoError(t, err)
			fixture := &remoteFixture{mode: mode, publicKey: string(identity.PublicKey), connected: make(chan struct{})}
			grpcServer := grpc.NewServer()
			grpcServer.RegisterService(&grpc.ServiceDesc{ServiceName: "databricks.sandbox.SandboxService", HandlerType: (*execService)(nil), Streams: []grpc.StreamDesc{{StreamName: "ExecuteCommand", ServerStreams: true, Handler: fixture.execute}}, Methods: []grpc.MethodDesc{{MethodName: "TerminateCommand", Handler: fixture.terminate}}}, fixture)
			defer grpcServer.Stop()
			endpoint := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
					grpcServer.ServeHTTP(w, r)
					return
				}
				fixture.serveHTTP(w, r)
			}))
			endpoint.EnableHTTP2 = true
			endpoint.StartTLS()
			defer endpoint.Close()
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: endpoint.URL, Token: "test-token", WorkspaceID: "123", InsecureSkipVerify: true, HTTPTransport: endpoint.Client().Transport})
			require.NoError(t, err)
			filename := "databricks_cli_" + build.GetInfo().Version + "_linux_amd64.zip"
			if build.GetInfo().IsDevelopment() {
				filename = "databricks_cli_linux_amd64.zip"
			}
			var buffer bytes.Buffer
			archive := zip.NewWriter(&buffer)
			file, err := archive.Create("databricks")
			require.NoError(t, err)
			_, err = file.Write([]byte("executable"))
			require.NoError(t, err)
			require.NoError(t, archive.Close())
			require.NoError(t, os.WriteFile(filepath.Join(home, filename), buffer.Bytes(), 0o600))
			if mode == "cancel" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Second)
				defer cancel()
			}
			input, writer := io.Pipe()
			defer input.Close()
			defer writer.Close()
			started := time.Now()
			startupTimeout := 10 * time.Minute
			if mode == "startup-timeout" || mode == "slow-session" {
				startupTimeout = time.Second
			}
			err = sandbox.ConnectProxyForTest(ctx, w, "sandboxes/test", identity, home, input, io.Discard, io.Discard, startupTimeout)
			if mode == "clean-close" || mode == "slow-session" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			switch mode {
			case "before-ready":
				assert.Contains(t, err.Error(), "exited before readiness")
			case "after-ready":
				assert.Contains(t, err.Error(), "server stopped")
			case "cancel":
				assert.ErrorIs(t, err, context.DeadlineExceeded)
			case "startup-timeout":
				assert.ErrorIs(t, err, context.DeadlineExceeded)
				assert.Less(t, time.Since(started), 5*time.Second)
			case "slow-session":
				assert.Greater(t, time.Since(started), startupTimeout)
			}
			fixture.mu.Lock()
			defer fixture.mu.Unlock()
			if mode == "after-ready" || mode == "clean-close" {
				assert.True(t, fixture.authenticated)
			}
			assert.True(t, fixture.uploaded)
			assert.True(t, fixture.terminated, "server command must be terminated independently of stream cancellation")
			assert.True(t, fixture.removed, "session files must be removed on failures")
		})
	}
}
