package sandbox_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if endpoint := os.Getenv("DATABRICKS_TEST_SANDBOX_PROXY_URL"); endpoint != "" {
		// TestMain also runs as a standalone SSH transport subprocess.
		//nolint:gocritic // TestMain has no testing.T context.
		err := sandbox.ProxyStreamForTest(context.Background(), os.Stdin, os.Stdout, true, func(ctx context.Context, request proxy.DialRequest) (*websocket.Conn, error) {
			conn, err := dialWebSocket(ctx, resumeEndpoint(endpoint, request))
			return conn, err
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if os.Getenv("DATABRICKS_TEST_SANDBOX_SSHD") == "1" {
		if os.Args[1] == "-t" {
			os.Exit(0)
		}
		if os.Getenv("DATABRICKS_TEST_SANDBOX_SSHD_PID") == "1" {
			fmt.Fprintf(os.Stdout, "pid:%d\n", os.Getpid())
		}
		_, _ = io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func startServer(t *testing.T, idle time.Duration) (string, context.CancelFunc, <-chan error, net.Listener) {
	t.Helper()
	t.Setenv("DATABRICKS_TEST_SANDBOX_SSHD", "1")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	reader, writer := io.Pipe()
	finished := make(chan error, 1)
	go func() {
		finished <- sandbox.Serve(ctx, listener, sandbox.ServerOptions{SSHD: executable, Config: "test-config", ConnectTimeout: idle}, writer)
		writer.Close()
	}()
	var ready sandbox.Readiness
	require.NoError(t, json.NewDecoder(reader).Decode(&ready))
	require.Equal(t, listener.Addr().(*net.TCPAddr).Port, ready.Port)
	require.Equal(t, proxy.ResumeProtocolVersion, ready.ResumeVersion)
	return "ws://token:" + ready.Token + "@" + listener.Addr().String() + "/ssh", cancel, finished, listener
}

func TestServerRelaysBinaryDataAndCancels(t *testing.T) {
	endpoint, cancel, finished, _ := startServer(t, time.Minute)
	conn, err := dialWebSocket(t.Context(), endpoint+"?id=first")
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	payload := bytes.Repeat([]byte{0, 1, 127, 128, 255}, 1000)
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, payload))
	var received []byte
	for len(received) < len(payload) {
		kind, chunk, err := conn.ReadMessage()
		require.NoError(t, err)
		require.Equal(t, websocket.BinaryMessage, kind)
		received = append(received, chunk...)
	}
	assert.Equal(t, payload, received)
	second, response, err := testDial(t.Context(), endpoint+"?id=second")
	if second != nil {
		second.Close()
	}
	require.Error(t, err)
	require.NotNil(t, response)
	assert.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	response.Body.Close()
	cancel()
	select {
	case err := <-finished:
		assert.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not reap its child on cancellation")
	}
}

func TestServerIdleShutdown(t *testing.T) {
	_, _, finished, _ := startServer(t, 50*time.Millisecond)
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("idle server did not stop")
	}
}

func TestCommandHeartbeatDoesNotPreventIdleShutdown(t *testing.T) {
	t.Setenv("DATABRICKS_TEST_SANDBOX_SSHD", "1")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	t.Cleanup(func() { writer.Close() })
	finished := make(chan error, 1)
	go func() {
		finished <- sandbox.Serve(ctx, listener, sandbox.ServerOptions{SSHD: executable, Config: "test-config", ConnectTimeout: time.Second, OutputHeartbeat: 10 * time.Millisecond}, writer)
	}()
	var ready sandbox.Readiness
	require.NoError(t, json.NewDecoder(reader).Decode(&ready))
	var newline [1]byte
	_, err = io.ReadFull(reader, newline[:])
	require.NoError(t, err)
	assert.Equal(t, byte('\n'), newline[0])
	// Stop draining output: the next heartbeat blocks, but idle shutdown must not.
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("stalled command output prevented idle shutdown")
	}
	_, err = writer.Write([]byte("after shutdown"))
	require.ErrorIs(t, err, io.ErrClosedPipe)
}

func TestServerRejectsInvalidConfigurationBeforeReadiness(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	var output bytes.Buffer
	err = sandbox.Serve(t.Context(), listener, sandbox.ServerOptions{SSHD: "missing-sshd", Config: "config", ConnectTimeout: time.Minute}, &output)
	require.ErrorContains(t, err, "validate sshd configuration")
	assert.Empty(t, output.String())
	_, err = listener.Accept()
	assert.ErrorIs(t, err, net.ErrClosed)
}

func TestServerReportsSSHDStartFailureAfterReadiness(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sandbox servers run on Linux")
	}
	sshd := filepath.Join(t.TempDir(), "sshd")
	require.NoError(t, os.WriteFile(sshd, []byte("#!/bin/sh\nexit 0\n"), 0o700))
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	reader, writer := io.Pipe()
	defer reader.Close()
	finished := make(chan error, 1)
	go func() {
		finished <- sandbox.Serve(ctx, listener, sandbox.ServerOptions{SSHD: sshd, Config: "config", ConnectTimeout: time.Minute}, writer)
		writer.Close()
	}()
	var ready sandbox.Readiness
	require.NoError(t, json.NewDecoder(reader).Decode(&ready))
	// Force a real exec failure after the configuration check succeeded.
	require.NoError(t, os.Remove(sshd))
	var clients sync.WaitGroup
	for i := range 16 {
		clients.Go(func() {
			endpoint := fmt.Sprintf("ws://token:%s@%s/ssh?id=failed-start-%d", ready.Token, listener.Addr(), i)
			conn, err := dialWebSocket(t.Context(), endpoint)
			if err == nil {
				_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
				_, _, _ = conn.ReadMessage()
				conn.Close()
			}
		})
	}
	select {
	case err := <-finished:
		require.ErrorContains(t, err, "failed to start SSHD process")
	case <-time.After(5 * time.Second):
		t.Fatal("sshd startup failure did not stop the server")
	}
	clients.Wait()
}

func TestServerStopsAfterItsSessionEnds(t *testing.T) {
	endpoint, _, finished, _ := startServer(t, time.Minute)
	first, err := dialWebSocket(t.Context(), endpoint+"?id=first")
	require.NoError(t, err)
	first.Close()
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop after its SSH session ended")
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken output") }

func TestServerReadinessWriteFailure(t *testing.T) {
	t.Setenv("DATABRICKS_TEST_SANDBOX_SSHD", "1")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	err = sandbox.Serve(t.Context(), listener, sandbox.ServerOptions{SSHD: executable, Config: "test-config", ConnectTimeout: time.Minute}, failingWriter{})
	require.ErrorContains(t, err, "report relay readiness")
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if conn != nil {
		conn.Close()
	}
	require.Error(t, err)
}

func TestServerListenerFailure(t *testing.T) {
	_, _, finished, listener := startServer(t, time.Minute)
	require.NoError(t, listener.Close())
	select {
	case err := <-finished:
		require.ErrorIs(t, err, net.ErrClosed)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not return after listener failure")
	}
}

func TestServerRejectsUnsupportedResumeProtocol(t *testing.T) {
	for _, query := range []string{"?id=first&delivered=0&resume_version=1", "?id=first&reattach=1"} {
		t.Run(query, func(t *testing.T) {
			endpoint, _, _, _ := startServer(t, time.Minute)
			conn, response, err := testDial(t.Context(), endpoint+query)
			if conn != nil {
				conn.Close()
			}
			require.Error(t, err)
			require.NotNil(t, response)
			defer response.Body.Close()
			assert.Equal(t, http.StatusBadRequest, response.StatusCode)

			conn, err = dialWebSocket(t.Context(), endpoint+"?id=plain")
			require.NoError(t, err)
			conn.Close()
		})
	}
}

func TestServerRejectsResumeOfMissingSession(t *testing.T) {
	endpoint, _, _, _ := startServer(t, time.Minute)
	conn, response, err := testDial(t.Context(), resumeEndpoint(endpoint, proxy.DialRequest{ConnID: "missing", ResumeCapable: true, Reattach: true}))
	if conn != nil {
		conn.Close()
	}
	require.Error(t, err)
	require.NotNil(t, response)
	defer response.Body.Close()
	assert.Equal(t, http.StatusGone, response.StatusCode)
}

func resumeEndpoint(endpoint string, request proxy.DialRequest) string {
	query := url.Values{"id": {request.ConnID}}
	if request.ResumeCapable {
		query.Set("delivered", strconv.FormatInt(request.Delivered, 10))
		query.Set(proxy.ResumeVersionParameter, strconv.Itoa(proxy.ResumeProtocolVersion))
	}
	if request.Reattach {
		query.Set("reattach", "1")
	}
	return endpoint + "?" + query.Encode()
}

func TestServerResumesSameProcessAfterWebSocketDrops(t *testing.T) {
	t.Setenv("DATABRICKS_TEST_SANDBOX_SSHD_PID", "1")
	endpoint, cancel, finished, _ := startServer(t, time.Minute)
	defer cancel()
	input, send := io.Pipe()
	receive, output := io.Pipe()
	defer send.Close()
	defer receive.Close()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	connections := make(chan *websocket.Conn, 8)
	var reattachments atomic.Int32
	result := make(chan error, 1)
	go func() {
		result <- proxy.RunClientProxy(ctx, input, output, func() <-chan time.Time { return nil }, time.Second, true, func(ctx context.Context, request proxy.DialRequest) (*websocket.Conn, error) {
			conn, err := dialWebSocket(ctx, resumeEndpoint(endpoint, request))
			if err == nil {
				connections <- conn
				if request.Reattach {
					reattachments.Add(1)
				}
			}
			return conn, err
		})
		output.Close()
	}()
	reader := bufio.NewReader(receive)
	pid, err := reader.ReadString('\n')
	require.NoError(t, err)
	require.Regexp(t, `^pid:[0-9]+\n$`, pid)
	for round := range 3 {
		var connection *websocket.Conn
		select {
		case connection = <-connections:
		case <-time.After(5 * time.Second):
			t.Fatal("client did not connect")
		}
		payload := bytes.Repeat([]byte{byte(round), 0, 127, 128, 255}, 10000)
		written := make(chan error, 1)
		go func() { _, err := send.Write(payload); written <- err }()
		require.NoError(t, connection.UnderlyingConn().Close())
		got := make([]byte, len(payload))
		read := make(chan error, 1)
		go func() { _, err := io.ReadFull(reader, got); read <- err }()
		select {
		case err := <-read:
			require.NoError(t, err)
		case <-time.After(10 * time.Second):
			t.Fatal("SSH output did not resume after the WebSocket dropped")
		}
		require.NoError(t, <-written)
		// A restarted child emits another PID line and corrupts this comparison.
		require.Equal(t, payload, got)
	}
	require.Equal(t, int32(3), reattachments.Load())
	stop()
	require.NoError(t, <-result)
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop after its resumed session ended")
	}
}

func TestServerKeepsActiveConnectionPastIdleTimeout(t *testing.T) {
	endpoint, cancel, finished, _ := startServer(t, 200*time.Millisecond)
	conn, err := dialWebSocket(t.Context(), endpoint+"?id=active")
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("before")))
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	_, _, err = conn.ReadMessage()
	require.NoError(t, err)
	time.Sleep(300 * time.Millisecond)
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("after")))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, "after", string(message))
	cancel()
	require.ErrorIs(t, <-finished, context.Canceled)
}

func dialWebSocket(ctx context.Context, endpoint string) (*websocket.Conn, error) {
	conn, response, err := testDial(ctx, endpoint)
	if response != nil {
		response.Body.Close()
	}
	return conn, err
}

// Blocking readiness past ConnectTimeout catches a deadline that starts before readiness is written.
type blockedReadiness struct {
	entered chan struct{}
	release chan struct{}
}

type failingHeartbeat struct {
	ready bool
	err   error
}

func (output *failingHeartbeat) Write(data []byte) (int, error) {
	if !output.ready {
		output.ready = true
		return len(data), nil
	}
	return 0, output.err
}

func (*failingHeartbeat) Close() error { return nil }

func TestServerStopsWhenCommandHeartbeatFails(t *testing.T) {
	t.Setenv("DATABRICKS_TEST_SANDBOX_SSHD", "1")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	brokenPipe := errors.New("command output pipe closed")
	output := &failingHeartbeat{err: brokenPipe}
	finished := make(chan error, 1)
	go func() {
		finished <- sandbox.Serve(ctx, listener, sandbox.ServerOptions{
			SSHD: executable, Config: "test-config", ConnectTimeout: time.Minute, OutputHeartbeat: 10 * time.Millisecond,
		}, output)
	}()
	select {
	case err := <-finished:
		require.ErrorIs(t, err, brokenPipe)
		require.ErrorContains(t, err, "write command heartbeat")
	case <-time.After(5 * time.Second):
		t.Fatal("server ignored its broken command-output pipe")
	}
	connection, err := net.DialTimeout("tcp4", listener.Addr().String(), time.Second)
	if connection != nil {
		connection.Close()
	}
	require.Error(t, err, "server listener must close after the heartbeat fails")
}

func (writer blockedReadiness) Write(data []byte) (int, error) {
	close(writer.entered)
	<-writer.release
	return len(data), nil
}

func TestServerConnectionDeadlineStartsAfterReadiness(t *testing.T) {
	t.Setenv("DATABRICKS_TEST_SANDBOX_SSHD", "1")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	output := blockedReadiness{entered: make(chan struct{}), release: make(chan struct{})}
	finished := make(chan error, 1)
	go func() {
		finished <- sandbox.Serve(ctx, listener, sandbox.ServerOptions{SSHD: executable, Config: "config", ConnectTimeout: 50 * time.Millisecond}, output)
	}()
	<-output.entered
	time.Sleep(100 * time.Millisecond)
	close(output.release)
	select {
	case err := <-finished:
		require.NoError(t, err)
		t.Fatal("connection deadline began before readiness")
	case <-time.After(20 * time.Millisecond):
	}
	require.NoError(t, <-finished)
}

func testDial(ctx context.Context, endpoint string) (*websocket.Conn, *http.Response, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, nil, err
	}
	headers := http.Header{}
	if u.User != nil {
		token, _ := u.User.Password()
		headers.Set(sandbox.RelayTokenHeader, token)
		u.User = nil
	}
	return websocket.DefaultDialer.DialContext(ctx, u.String(), headers)
}

func TestServerRejectsConnectionsWithoutInvocationToken(t *testing.T) {
	endpoint, _, finished, _ := startServer(t, 100*time.Millisecond)
	u, err := url.Parse(endpoint)
	require.NoError(t, err)
	u.User = nil
	for _, token := range []string{"", "another-invocation"} {
		headers := http.Header{}
		headers.Set(sandbox.RelayTokenHeader, token)
		conn, response, err := websocket.DefaultDialer.DialContext(t.Context(), u.String()+"?id=intruder", headers)
		if conn != nil {
			conn.Close()
		}
		require.Error(t, err)
		require.NotNil(t, response)
		assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
		response.Body.Close()
	}
	require.NoError(t, <-finished)
}

func TestServerMalformedRequestsDoNotExtendConnectionDeadline(t *testing.T) {
	endpoint, _, finished, _ := startServer(t, 100*time.Millisecond)
	deadline := time.After(3 * time.Second)
	ticker := time.NewTicker(5 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-finished:
			require.NoError(t, err)
			return
		case <-deadline:
			t.Fatal("malformed requests kept the server alive")
		case <-ticker.C:
			conn, response, _ := testDial(t.Context(), endpoint)
			if conn != nil {
				conn.Close()
			}
			if response != nil {
				response.Body.Close()
			}
		}
	}
}

func TestServerHandoverSurvivesWebSocketLifetimeLimit(t *testing.T) {
	endpoint, cancel, finished, _ := startServer(t, time.Minute)
	defer cancel()
	target, err := url.Parse(endpoint)
	require.NoError(t, err)
	token, _ := target.User.Password()
	target.User = nil
	target.Scheme = "http"
	target.Path = ""
	forwarding := httptest.NewUnstartedServer(httputil.NewSingleHostReverseProxy(target))
	forwarding.Config.ConnState = func(conn net.Conn, state http.ConnState) {
		if state == http.StateHijacked {
			time.AfterFunc(500*time.Millisecond, func() { conn.Close() })
		}
	}
	forwarding.Start()
	defer forwarding.Close()
	input, send := io.Pipe()
	receive, output := io.Pipe()
	defer send.Close()
	defer receive.Close()
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()
	var dials atomic.Int32
	result := make(chan error, 1)
	go func() {
		result <- proxy.RunClientProxy(ctx, input, output, func() <-chan time.Time { return tick.C }, time.Second, false,
			func(ctx context.Context, request proxy.DialRequest) (*websocket.Conn, error) {
				headers := http.Header{}
				headers.Set(sandbox.RelayTokenHeader, token)
				conn, response, err := websocket.DefaultDialer.DialContext(ctx, "ws"+strings.TrimPrefix(forwarding.URL, "http")+"/ssh?id="+request.ConnID, headers)
				if response != nil {
					response.Body.Close()
				}
				if err == nil {
					dials.Add(1)
				}
				return conn, err
			})
	}()
	for i := range 5 {
		payload := []byte(fmt.Sprintf("message-%d", i))
		written := make(chan error, 1)
		go func() { _, err := send.Write(payload); written <- err }()
		select {
		case err := <-written:
			require.NoError(t, err)
		case err := <-result:
			t.Fatalf("client ended before data transfer: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("client did not read SSH input")
		}
		got := make([]byte, len(payload))
		read := make(chan error, 1)
		go func() { _, err := io.ReadFull(receive, got); read <- err }()
		select {
		case err := <-read:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Fatal("SSH stream stopped during handover")
		}
		require.Equal(t, payload, got)
		time.Sleep(200 * time.Millisecond)
	}
	require.GreaterOrEqual(t, dials.Load(), int32(3))
	stop()
	require.NoError(t, <-result)
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop after its handed-over session ended")
	}
}

func TestServerFailedUpgradeCanRetry(t *testing.T) {
	endpoint, _, finished, _ := startServer(t, time.Minute)
	u, err := url.Parse(endpoint)
	require.NoError(t, err)
	token, _ := u.User.Password()
	u.User = nil
	u.Scheme = "http"
	u.RawQuery = "id=failed"
	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u.String(), nil)
	require.NoError(t, err)
	request.Header.Set(sandbox.RelayTokenHeader, token)
	response, err := http.DefaultClient.Do(request)
	require.NoError(t, err)
	response.Body.Close()
	require.Equal(t, http.StatusBadRequest, response.StatusCode)

	conn, err := dialWebSocket(t.Context(), endpoint+"?id=retry")
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("retry")))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, "retry", string(message))
	require.NoError(t, conn.Close())
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop after the retried session ended")
	}
}
