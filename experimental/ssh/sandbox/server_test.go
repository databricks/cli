package sandbox_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/databricks/cli/experimental/ssh/internal/proxy"
	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	if os.Getenv("DATABRICKS_TEST_SANDBOX_SSHD") == "1" {
		if os.Args[1] == "-t" {
			os.Exit(0)
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
		finished <- sandbox.Serve(ctx, listener, sandbox.ServerOptions{SSHD: executable, Config: "test-config", IdleTimeout: idle}, writer)
		writer.Close()
	}()
	var ready sandbox.Readiness
	require.NoError(t, json.NewDecoder(reader).Decode(&ready))
	require.Equal(t, listener.Addr().(*net.TCPAddr).Port, ready.Port)
	return "ws://" + listener.Addr().String() + "/ssh", cancel, finished, listener
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
	second, response, err := websocket.DefaultDialer.DialContext(t.Context(), endpoint+"?id=second", nil)
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

func TestServerRejectsInvalidConfigurationBeforeReadiness(t *testing.T) {
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	var output bytes.Buffer
	err = sandbox.Serve(t.Context(), listener, sandbox.ServerOptions{SSHD: "missing-sshd", Config: "config", IdleTimeout: time.Minute}, &output)
	require.ErrorContains(t, err, "validate sshd configuration")
	assert.Empty(t, output.String())
	_, err = listener.Accept()
	assert.ErrorIs(t, err, net.ErrClosed)
}

func TestServerReleasesConnectionSlot(t *testing.T) {
	endpoint, _, _, _ := startServer(t, time.Minute)
	first, err := dialWebSocket(t.Context(), endpoint+"?id=first")
	require.NoError(t, err)
	first.Close()
	var second *websocket.Conn
	require.Eventually(t, func() bool {
		conn, response, err := websocket.DefaultDialer.DialContext(t.Context(), endpoint+"?id=second", nil)
		if response != nil {
			response.Body.Close()
		}
		if err != nil {
			return false
		}
		second = conn
		return true
	}, 5*time.Second, 10*time.Millisecond)
	defer second.Close()
	require.NoError(t, second.SetReadDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, second.WriteMessage(websocket.BinaryMessage, []byte("second")))
	_, result, err := second.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, "second", string(result))
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("broken output") }

func TestServerReadinessWriteFailure(t *testing.T) {
	t.Setenv("DATABRICKS_TEST_SANDBOX_SSHD", "1")
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	require.NoError(t, err)
	executable, err := os.Executable()
	require.NoError(t, err)
	err = sandbox.Serve(t.Context(), listener, sandbox.ServerOptions{SSHD: executable, Config: "test-config", IdleTimeout: time.Minute}, failingWriter{})
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

func TestServerRejectsResumeProtocol(t *testing.T) {
	resume := fmt.Sprintf("?id=first&delivered=0&resume_version=%d", proxy.ResumeProtocolVersion)
	for _, query := range []string{resume, resume + "&reattach=1"} {
		t.Run(query, func(t *testing.T) {
			endpoint, _, _, _ := startServer(t, time.Minute)
			conn, response, err := websocket.DefaultDialer.DialContext(t.Context(), endpoint+query, nil)
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
	conn, response, err := websocket.DefaultDialer.DialContext(ctx, endpoint, nil)
	if response != nil {
		response.Body.Close()
	}
	return conn, err
}

// Blocking readiness lets the idle timer fire before Serve starts receiving it.
type blockedReadiness struct {
	entered chan struct{}
	release chan struct{}
}

func (writer blockedReadiness) Write(data []byte) (int, error) {
	close(writer.entered)
	<-writer.release
	return len(data), nil
}

func TestServerIgnoresQueuedIdleTimeoutAndRestartsItAfterDisconnect(t *testing.T) {
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
		finished <- sandbox.Serve(ctx, listener, sandbox.ServerOptions{SSHD: executable, Config: "config", IdleTimeout: 50 * time.Millisecond}, output)
	}()
	<-output.entered
	time.Sleep(100 * time.Millisecond)
	conn, err := dialWebSocket(t.Context(), "ws://"+listener.Addr().String()+"/ssh?id=queued")
	close(output.release)
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(5*time.Second)))
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("still active")))
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	assert.Equal(t, "still active", string(message))
	conn.Close()
	select {
	case err := <-finished:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("idle timeout was not restarted after disconnect")
	}
}
