package sandbox_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/databricks/cli/experimental/ssh/sandbox/internal/sandboxexec"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type execService any

type remoteFixture struct {
	mode                           string
	forwardStatus, forwardFailures int
	terminateError                 bool
	resumeVersion                  int
	publicKey                      string
	connected                      chan struct{}
	outputEnded                    chan struct{}
	mu                             sync.Mutex
	terminated, removed, uploaded  bool
	authenticated                  bool
	commandID                      string
	dialQueries                    []url.Values
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
	case "after-ready", "clean-close", "slow-session", "clean-output-first", "resume-dial", "resume-gone", "resume-conflict":
		readiness := "{\"port\":40000,\"token\":\"invocation-token\"}\n"
		if fixture.resumeVersion != 0 {
			readiness = fmt.Sprintf("{\"port\":40000,\"token\":\"invocation-token\",\"resume_version\":%d}\n", fixture.resumeVersion)
		}
		if err := stream.SendMsg(&sandboxexec.ExecuteCommandResponse{ProcessEvent: &sandboxexec.ProcessEvent{Event: &sandboxexec.ProcessEvent_Data{Data: &sandboxexec.DataEvent{Output: &sandboxexec.DataEvent_Stdout{Stdout: []byte(readiness)}}}}}); err != nil {
			return err
		}
		select {
		case <-fixture.connected:
		case <-stream.Context().Done():
			return stream.Context().Err()
		}
		if fixture.mode == "clean-close" || fixture.mode == "slow-session" || fixture.mode == "resume-gone" || fixture.mode == "resume-conflict" {
			<-stream.Context().Done()
			return stream.Context().Err()
		}
	}
	if fixture.mode == "clean-output-first" {
		err := stream.SendMsg(&sandboxexec.ExecuteCommandResponse{ProcessEvent: &sandboxexec.ProcessEvent{Event: &sandboxexec.ProcessEvent_End{End: &sandboxexec.EndEvent{ExitCode: proto.Int32(0)}}}})
		close(fixture.outputEnded)
		return err
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
	if fixture.terminateError {
		return nil, status.Error(codes.Unavailable, "termination failed")
	}
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
			if len(request.Args) > 2 && (request.Args[2] == "check" || request.Args[2] == "install") {
				result["stdout"] = fmt.Sprintf(`{"cached":%t}`, request.Args[2] == "install")
				break
			}
			setup, _ := json.Marshal(map[string]string{"home": "/home/sandbox-agent", "config": "/home/sandbox-agent/session/sshd_config", "sshd": "/usr/sbin/sshd", "user": "sandbox-agent", "host_key": fixture.publicKey, "session_dir": "/home/sandbox-agent/session", "architecture": "x86_64"})
			result["stdout"] = string(setup)
		case "rm":
			fixture.mu.Lock()
			fixture.removed = true
			fixture.mu.Unlock()
		}
		_ = json.NewEncoder(w).Encode(result)
	case strings.HasPrefix(r.URL.Path, "/sandbox-proxy/"):
		if r.Header.Get(sandbox.RelayTokenHeader) != "invocation-token" || r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-Databricks-Workspace-Id") != "123" {
			http.Error(w, "missing forwarding credentials", http.StatusUnauthorized)
			return
		}
		fixture.mu.Lock()
		fixture.authenticated = true
		fixture.dialQueries = append(fixture.dialQueries, r.URL.Query())
		attempt := len(fixture.dialQueries)
		fixture.mu.Unlock()
		if attempt <= fixture.forwardFailures {
			http.Error(w, "forwarding rejected", fixture.forwardStatus)
			return
		}
		if fixture.mode == "resume-dial" {
			http.Error(w, "connection rejected", http.StatusGone)
			return
		}
		if r.URL.Query().Get("reattach") == "1" {
			switch fixture.mode {
			case "resume-gone":
				http.Error(w, "session expired", http.StatusGone)
				return
			case "resume-conflict":
				http.Error(w, "session unavailable", http.StatusConflict)
				return
			}
		}
		upgrader := websocket.Upgrader{}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		_ = conn.WriteMessage(websocket.BinaryMessage, []byte("SSH-2.0-test\r\n"))
		close(fixture.connected)
		if fixture.mode == "resume-gone" || fixture.mode == "resume-conflict" {
			return
		}
		if fixture.mode == "slow-session" {
			time.Sleep(1500 * time.Millisecond)
		}
		if fixture.mode == "clean-output-first" {
			<-fixture.outputEnded
			// Deliver the clean command exit before closing the forwarding connection.
			time.Sleep(200 * time.Millisecond)
		}
		if fixture.mode == "clean-close" || fixture.mode == "slow-session" || fixture.mode == "clean-output-first" {
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
	for _, tc := range []struct {
		name, mode                     string
		resumeVersion                  int
		quiet                          bool
		forwardStatus, forwardFailures int
		wantForwardError               bool
		terminateError                 bool
	}{
		{name: "forwarding-503", mode: "clean-close", forwardStatus: http.StatusServiceUnavailable, forwardFailures: 2},
		{name: "forwarding-429", mode: "clean-close", forwardStatus: http.StatusTooManyRequests, forwardFailures: 1},
		{name: "forwarding-503-stop-failed", mode: "clean-close", forwardStatus: http.StatusServiceUnavailable, forwardFailures: 1, wantForwardError: true, terminateError: true},
		{name: "forwarding-403", mode: "clean-close", forwardStatus: http.StatusForbidden, forwardFailures: 1, wantForwardError: true},
		{name: "before-ready", mode: "before-ready"},
		{name: "after-ready", mode: "after-ready"},
		{name: "cancel", mode: "cancel"},
		{name: "clean-close", mode: "clean-close"},
		{name: "quiet-clean-close", mode: "clean-close", quiet: true},
		{name: "startup-timeout", mode: "startup-timeout"},
		{name: "slow-session", mode: "slow-session"},
		{name: "clean-output-first", mode: "clean-output-first"},
		{name: "unsupported-resume-v1", mode: "clean-close", resumeVersion: 1},
		{name: "unsupported-resume-future", mode: "clean-close", resumeVersion: proxy.ResumeProtocolVersion + 1},
		{name: "resume-initial-dial", mode: "resume-dial", resumeVersion: proxy.ResumeProtocolVersion},
		{name: "resume-reattach-gone", mode: "resume-gone", resumeVersion: proxy.ResumeProtocolVersion},
		{name: "resume-reattach-conflict", mode: "resume-conflict", resumeVersion: proxy.ResumeProtocolVersion},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mode := tc.mode
			home := t.TempDir()
			ctx := env.WithUserHomeDir(t.Context(), home)
			if tc.quiet {
				ctx = cmdio.WithQuiet(ctx)
			}
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			identity, err := sandbox.LocalIdentity(ctx, "https://workspace.test", "caller", "sandboxes/test")
			require.NoError(t, err)
			fixture := &remoteFixture{mode: mode, terminateError: tc.terminateError, forwardStatus: tc.forwardStatus, forwardFailures: tc.forwardFailures, resumeVersion: tc.resumeVersion, publicKey: string(identity.PublicKey), connected: make(chan struct{}), outputEnded: make(chan struct{})}
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
				ctx, cancel = context.WithCancel(ctx)
				defer cancel()
				timer := time.AfterFunc(time.Second, cancel)
				defer timer.Stop()
			}
			input, writer := io.Pipe()
			defer input.Close()
			defer writer.Close()
			started := time.Now()
			startupTimeout := 10 * time.Minute
			if mode == "startup-timeout" || mode == "slow-session" {
				startupTimeout = time.Second
			}
			var diagnostics bytes.Buffer
			err = sandbox.ConnectProxyForTest(ctx, w, "sandboxes/test", identity, home, input, io.Discard, &diagnostics, startupTimeout)
			if (mode == "clean-close" || mode == "slow-session" || mode == "clean-output-first") && !tc.wantForwardError {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
			if tc.wantForwardError {
				assert.ErrorContains(t, err, fmt.Sprintf("HTTP %d", tc.forwardStatus))
			}
			switch mode {
			case "before-ready":
				assert.Contains(t, err.Error(), "exited before readiness")
			case "after-ready":
				assert.Contains(t, err.Error(), "server stopped")
			case "cancel":
				assert.ErrorIs(t, err, context.Canceled)
			case "startup-timeout":
				assert.ErrorIs(t, err, context.DeadlineExceeded)
				assert.Less(t, time.Since(started), 5*time.Second)
			case "slow-session":
				assert.Greater(t, time.Since(started), startupTimeout)
			case "resume-dial":
				assert.ErrorIs(t, err, proxy.ErrConnectFailed)
				assert.NotErrorIs(t, err, proxy.ErrReattachRejected)
			case "resume-gone", "resume-conflict":
				assert.ErrorIs(t, err, proxy.ErrWebsocketDropped)
				assert.ErrorIs(t, err, proxy.ErrReattachRejected)
			}
			fixture.mu.Lock()
			defer fixture.mu.Unlock()
			if mode == "after-ready" || mode == "clean-close" || tc.resumeVersion == proxy.ResumeProtocolVersion {
				assert.True(t, fixture.authenticated)
			}
			if fixture.authenticated {
				if mode == "resume-gone" || mode == "resume-conflict" {
					require.Len(t, fixture.dialQueries, 2)
					assert.Equal(t, url.Values{
						"id":                         {fixture.dialQueries[0].Get("id")},
						proxy.ResumeVersionParameter: {"2"},
						"delivered":                  {strconv.Itoa(len("SSH-2.0-test\r\n"))},
						"reattach":                   {"1"},
					}, fixture.dialQueries[1])
				} else {
					expected := 1
					if !tc.wantForwardError {
						expected += tc.forwardFailures
					}
					require.Len(t, fixture.dialQueries, expected)
					for i, query := range fixture.dialQueries[1:] {
						if tc.forwardFailures > 0 {
							assert.NotEqual(t, fixture.dialQueries[i].Get("id"), query.Get("id"))
						} else {
							assert.Equal(t, fixture.dialQueries[0], query)
						}
					}
				}
				query := fixture.dialQueries[0]
				assert.NotEmpty(t, query.Get("id"))
				assert.False(t, query.Has("reattach"))
				if tc.resumeVersion == proxy.ResumeProtocolVersion {
					assert.Equal(t, "2", query.Get(proxy.ResumeVersionParameter))
					assert.Equal(t, "0", query.Get("delivered"))
				} else {
					assert.False(t, query.Has(proxy.ResumeVersionParameter))
					assert.False(t, query.Has("delivered"))
				}
			}
			assert.True(t, fixture.uploaded)
			assert.True(t, fixture.terminated, "server command must be terminated independently of stream cancellation")
			remove := err == nil || mode == "cancel"
			assert.Equal(t, remove, fixture.removed, "preserve session logs on failures; clean up successful or cancelled sessions")
			if !remove {
				assert.Contains(t, diagnostics.String(), "session files retained at /home/sandbox-agent/session")
			}
			if tc.quiet {
				assert.NotContains(t, diagnostics.String(), "Waiting for")
				assert.NotContains(t, diagnostics.String(), "checking or uploading")
			} else {
				assert.Contains(t, diagnostics.String(), "Waiting for sandboxes/test to be running")
				assert.Contains(t, diagnostics.String(), "checking or uploading the CLI executable")
			}
		})
	}
}
