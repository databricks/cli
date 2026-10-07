//go:build !windows

// TODO: figure out what command can we use on Windows for the echo server
package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestServer(t *testing.T, maxClients int, shutdownDelay time.Duration) *httptest.Server {
	ctx := cmdio.MockDiscard(t.Context())
	connections := NewConnectionsManager(maxClients, shutdownDelay)
	proxyServer := NewProxyServer(ctx, connections, func(ctx context.Context) *exec.Cmd {
		// 'cat' command reads each line from stdin and sends it to stdout, so we can test end-to-end proxying.
		// '-u' option is used to disable output buffering.
		return exec.CommandContext(ctx, "cat", "-u")
	})
	return httptest.NewServer(proxyServer)
}

type testClient struct {
	InputWriter *io.PipeWriter
	Output      *testBuffer
	// Done is closed when RunClientProxy returns.
	Done    <-chan struct{}
	Cleanup func()
}

func createTestClient(t *testing.T, serverURL string, requestHandoverTick func() <-chan time.Time, keepaliveInterval time.Duration, errChan chan error) *testClient {
	wsURL := "ws" + serverURL[4:]
	createConn := func(ctx context.Context, dial DialRequest) (*websocket.Conn, error) {
		url := fmt.Sprintf("%s?id=%s", wsURL, dial.ConnID)
		conn, _, err := websocket.DefaultDialer.Dial(url, nil) // nolint:bodyclose
		return conn, err
	}
	return createTestClientWithDialer(t, createConn, requestHandoverTick, keepaliveInterval, false, errChan)
}

// createTestClientWithDialer is createTestClient with the websocket dialer supplied by the caller,
// so a test can control which dials succeed - the initial connection's or a handover's.
func createTestClientWithDialer(t *testing.T, createConn createWebsocketConnectionFunc, requestHandoverTick func() <-chan time.Time, keepaliveInterval time.Duration, resumable bool, errChan chan error) *testClient {
	ctx := cmdio.MockDiscard(t.Context())
	clientInput, clientInputWriter := io.Pipe()
	clientOutput := newTestBuffer(t)
	if requestHandoverTick == nil {
		requestHandoverTick = neverTick
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := RunClientProxy(ctx, clientInput, clientOutput, requestHandoverTick, keepaliveInterval, resumable, createConn)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, io.ErrClosedPipe) {
			if errChan != nil {
				errChan <- err
			} else {
				t.Errorf("client error: %v", err)
			}
		}
	}()
	return &testClient{
		InputWriter: clientInputWriter,
		Output:      clientOutput,
		Done:        done,
		Cleanup: func() {
			clientInput.Close()
			clientInputWriter.Close()
			<-done
		},
	}
}

func TestClientServerEcho(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()
	client := createTestClient(t, server.URL, nil, time.Hour, nil)
	defer client.Cleanup()

	testMsg1 := []byte("test message 1\n")
	_, err := client.InputWriter.Write(testMsg1)
	require.NoError(t, err)
	err = client.Output.AssertWrite(testMsg1)
	require.NoError(t, err)

	testMsg2 := []byte("test message 2\n")
	_, err = client.InputWriter.Write(testMsg2)
	require.NoError(t, err)
	err = client.Output.AssertWrite(testMsg2)
	require.NoError(t, err)

	expectedOutput := fmt.Sprintf("%s%s", testMsg1, testMsg2)
	assert.Equal(t, expectedOutput, client.Output.String())
}

func TestMultipleClients(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()
	client1 := createTestClient(t, server.URL, nil, time.Hour, nil)
	defer client1.Cleanup()
	client2 := createTestClient(t, server.URL, nil, time.Hour, nil)
	defer client2.Cleanup()

	messageCount := 10
	var expectedClientOutput1, expectedClientOutput2 []byte
	for i := range messageCount {
		message := fmt.Appendf(nil, "client 1 message %d\n", i)
		_, err := client1.InputWriter.Write(message)
		require.NoError(t, err)
		err = client1.Output.AssertWrite(message)
		require.NoError(t, err)
		expectedClientOutput1 = append(expectedClientOutput1, message...)

		message = fmt.Appendf(nil, "client 2 message %d\n", i)
		_, err = client2.InputWriter.Write(message)
		require.NoError(t, err)
		err = client2.Output.AssertWrite(message)
		require.NoError(t, err)
		expectedClientOutput2 = append(expectedClientOutput2, message...)
	}

	assert.Equal(t, string(expectedClientOutput1), client1.Output.String())
	assert.Equal(t, string(expectedClientOutput2), client2.Output.String())
}

func TestMaxClients(t *testing.T) {
	maxClients := 2
	server := createTestServer(t, maxClients, time.Hour)
	defer server.Close()
	client1 := createTestClient(t, server.URL, nil, time.Hour, nil)
	defer client1.Cleanup()
	client2 := createTestClient(t, server.URL, nil, time.Hour, nil)
	defer client2.Cleanup()

	testMsg1 := []byte("test message 1\n")
	_, err := client1.InputWriter.Write(testMsg1)
	require.NoError(t, err)
	err = client1.Output.AssertWrite(testMsg1)
	require.NoError(t, err)
	_, err = client2.InputWriter.Write(testMsg1)
	require.NoError(t, err)
	err = client2.Output.AssertWrite(testMsg1)
	require.NoError(t, err)

	errChan := make(chan error, 1)
	client3 := createTestClient(t, server.URL, nil, time.Hour, errChan)
	defer client3.Cleanup()
	select {
	case err = <-errChan:
		require.Error(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("expected error due to max clients reached, but got none")
	}
}

func TestHandover(t *testing.T) {
	t.Run("without keepalive", func(t *testing.T) {
		runHandoverExchange(t, time.Hour)
	})
	// Pings and the data stream share the connection's write lock: they must not corrupt or reorder
	// the stream, nor trip gorilla's concurrent-write panic.
	t.Run("with keepalive", func(t *testing.T) {
		runHandoverExchange(t, time.Millisecond)
	})
}

func runHandoverExchange(t *testing.T, keepaliveInterval time.Duration) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	handoverChan := make(chan time.Time)
	requestHandoverTick := func() <-chan time.Time {
		return handoverChan
	}
	client := createTestClient(t, server.URL, requestHandoverTick, keepaliveInterval, nil)
	defer client.Cleanup()

	var expectedOutput []byte

	wg := sync.WaitGroup{}
	wg.Go(func() {
		for i := range TOTAL_MESSAGE_COUNT {
			if i > 0 && i%MESSAGES_PER_CHUNK == 0 && i < TOTAL_MESSAGE_COUNT-1 {
				handoverChan <- time.Now()
			}
			message := fmt.Appendf(nil, "message %d\n", i)
			_, err := client.InputWriter.Write(message)
			if err != nil {
				t.Errorf("failed to write message %d: %v", i, err)
			}
			expectedOutput = append(expectedOutput, message...)
		}
	})

	err := client.Output.WaitForWrite(fmt.Appendf(nil, "message %d\n", TOTAL_MESSAGE_COUNT-1))
	require.NoError(t, err, "failed to receive the last message (%d)", TOTAL_MESSAGE_COUNT-1)

	wg.Wait()

	// client.Output is created by appending incoming messages as they arrive, so we are also test correct order here
	assert.Equal(t, string(expectedOutput), client.Output.String())
}

// Tests handovers in quick succession with few messages in between.
// Not a real world scenario, but it can help uncover potential race conditions or deadlocks.
func TestQuickHandover(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	handoverChan := make(chan time.Time)
	requestHandoverTick := func() <-chan time.Time {
		return handoverChan
	}
	client := createTestClient(t, server.URL, requestHandoverTick, time.Hour, nil)
	defer client.Cleanup()

	var expectedOutput []byte

	wg := sync.WaitGroup{}
	wg.Go(func() {
		for i := range 16 {
			if i == 4 || i == 8 || i == 12 {
				handoverChan <- time.Now()
			}
			message := fmt.Appendf(nil, "message %d\n", i)
			_, err := client.InputWriter.Write(message)
			if err != nil {
				t.Errorf("failed to write message %d: %v", i, err)
			}
			expectedOutput = append(expectedOutput, message...)
		}
	})

	err := client.Output.WaitForWrite(fmt.Appendf(nil, "message %d\n", 15))
	require.NoError(t, err, "failed to receive the last message (%d)", 15)

	wg.Wait()

	assert.Equal(t, string(expectedOutput), client.Output.String())
}

// A handover that fails while dialing its replacement connection must not end the session: the
// connection it was meant to replace is still live and carrying traffic. Until this was handled,
// a single transient dial failure - a token refresh, a DNS blip, a proxy stumble - dropped an
// otherwise healthy session once every handover interval.
func TestHandoverDialFailureKeepsSessionAlive(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	var dials atomic.Int32
	// Signalled the instant a handover dial is attempted (and made to fail). initiateHandover
	// already holds handoverMutex by the time it dials, so a receive here proves the handover
	// goroutine has entered the dial and taken the mutex. Buffered and sent non-blockingly so the
	// dialer never stalls on it even if more dials than expected occur.
	handoverDialAttempted := make(chan struct{}, 1)
	createConn := func(ctx context.Context, dial DialRequest) (*websocket.Conn, error) {
		// Let the initial connection through and fail every handover dial after it.
		if dials.Add(1) > 1 {
			select {
			case handoverDialAttempted <- struct{}{}:
			default:
			}
			return nil, errors.New("simulated transient dial failure")
		}
		url := fmt.Sprintf("%s?id=%s", wsURL, dial.ConnID)
		conn, _, err := websocket.DefaultDialer.Dial(url, nil) // nolint:bodyclose
		return conn, err
	}

	handoverChan := make(chan time.Time)
	errChan := make(chan error, 1)
	client := createTestClientWithDialer(t, createConn, func() <-chan time.Time {
		return handoverChan
	}, time.Hour, false, errChan)
	defer client.Cleanup()

	beforeMsg := []byte("before handover\n")
	_, err := client.InputWriter.Write(beforeMsg)
	require.NoError(t, err)
	require.NoError(t, client.Output.AssertWrite(beforeMsg))

	handoverChan <- time.Now()

	// Completing the tick send only proves the handover goroutine received the tick; it does not
	// prove it acquired handoverMutex and reached the dial. Wait for the dial to actually be
	// attempted before sending more traffic - otherwise the payload below can traverse the
	// original connection before the handover even starts, which is the macOS "dials == 1" flake.
	select {
	case <-handoverDialAttempted:
	case <-time.After(10 * time.Second):
		t.Fatal("the handover never attempted its replacement dial")
	}

	// The original connection must still be proxying both ways. sendMessage blocks on the
	// handover mutex, so this write cannot overtake the failed handover.
	afterMsg := []byte("after failed handover\n")
	_, err = client.InputWriter.Write(afterMsg)
	require.NoError(t, err)
	require.NoError(t, client.Output.AssertWrite(afterMsg))

	select {
	case err := <-errChan:
		t.Fatalf("session ended after a failed handover dial: %v", err)
	default:
	}
	assert.Equal(t, int32(2), dials.Load(), "expected the initial dial plus exactly one handover dial")
}

// A failed dial never touches the live connection, whatever error it returns, so the handover loop
// must check for it before it treats context.Canceled as the session ending. Later ticks must still
// hand over.
func TestHandoverDialCanceledKeepsHandingOver(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	var dials atomic.Int32
	handoverDialed := make(chan struct{}, 2)
	createConn := func(ctx context.Context, dial DialRequest) (*websocket.Conn, error) {
		n := dials.Add(1)
		if n > 1 {
			handoverDialed <- struct{}{}
		}
		if n == 2 {
			return nil, context.Canceled
		}
		url := fmt.Sprintf("%s?id=%s", wsURL, dial.ConnID)
		conn, _, err := websocket.DefaultDialer.Dial(url, nil) // nolint:bodyclose
		return conn, err
	}

	handoverChan := make(chan time.Time)
	client := createTestClientWithDialer(t, createConn, func() <-chan time.Time {
		return handoverChan
	}, time.Hour, false, nil)
	defer client.Cleanup()

	for i := range 2 {
		select {
		case handoverChan <- time.Now():
		case <-time.After(10 * time.Second):
			t.Fatalf("handover %d was never started: the handover loop has stopped", i+1)
		}
		// The tick only proves the loop received it. Wait for the dial, which runs under the
		// handover mutex, so the write below cannot overtake the handover.
		select {
		case <-handoverDialed:
		case <-time.After(10 * time.Second):
			t.Fatalf("handover %d never dialed", i+1)
		}
		// sendMessage waits for the handover mutex, so this round trip also waits out the handover.
		msg := fmt.Appendf(nil, "after handover %d\n", i+1)
		_, err := client.InputWriter.Write(msg)
		require.NoError(t, err)
		require.NoError(t, client.Output.AssertWrite(msg))
	}
	assert.Equal(t, int32(3), dials.Load(), "expected the initial dial plus two handover dials")
}

// A session that ends while a handover is still waiting for the old connection to close must end
// with the session's own outcome, not ErrHandoverFailed, and must not start another handover.
func TestSessionEndDuringHandover(t *testing.T) {
	errSourceFailed := errors.New("source failed")
	tests := []struct {
		name       string
		resumable  bool
		endSession func(w *io.PipeWriter)
		wantErr    error
	}{
		{
			name:       "clean exit",
			endSession: func(w *io.PipeWriter) { w.Close() },
		},
		{
			name:       "source error",
			endSession: func(w *io.PipeWriter) { w.CloseWithError(errSourceFailed) },
			wantErr:    errSourceFailed,
		},
		{
			name:       "resumable clean exit",
			resumable:  true,
			endSession: func(w *io.PipeWriter) { w.Close() },
		},
		{
			name:       "resumable source error",
			resumable:  true,
			endSession: func(w *io.PipeWriter) { w.CloseWithError(errSourceFailed) },
			wantErr:    errSourceFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := createTestServer(t, 2, time.Hour)
			defer server.Close()
			// The handover dial lands here instead of on the proxy server, so nobody ever closes the
			// old connection and the handover stays in progress until the session ends.
			silentServer := blackHoleServer(t)
			defer silentServer.Close()

			wsURL := "ws" + server.URL[4:]
			silentURL := "ws" + silentServer.URL[4:]
			var dials atomic.Int32
			handoverDialed := make(chan struct{}, 1)
			createConn := func(ctx context.Context, dial DialRequest) (*websocket.Conn, error) {
				isHandover := dials.Add(1) > 1
				url := fmt.Sprintf("%s?id=%s", wsURL, dial.ConnID)
				if dial.ResumeCapable {
					url += fmt.Sprintf("&resume_version=2&delivered=%d", dial.Delivered)
				}
				if isHandover {
					url = silentURL
				}
				conn, _, err := websocket.DefaultDialer.Dial(url, nil) // nolint:bodyclose
				// Only a successful dial leaves the handover in progress.
				if isHandover && err == nil {
					select {
					case handoverDialed <- struct{}{}:
					default:
					}
				}
				return conn, err
			}

			// The second tick waits in the channel while the first handover is in progress, so a
			// handover loop that keeps going after the session ended dials again.
			ticks := make(chan time.Time, 2)
			ticks <- time.Now()
			ticks <- time.Now()
			errChan := make(chan error, 1)
			client := createTestClientWithDialer(t, createConn, func() <-chan time.Time {
				return ticks
			}, time.Hour, tt.resumable, errChan)
			defer client.Cleanup()

			select {
			case <-handoverDialed:
			case <-time.After(10 * time.Second):
				t.Fatal("the handover never dialed its replacement connection")
			}

			// Wait for the session to end before Cleanup closes the reader too: a read that
			// wakes up after that returns io.ErrClosedPipe instead of what endSession set.
			tt.endSession(client.InputWriter)
			select {
			case <-client.Done:
			case <-time.After(10 * time.Second):
				t.Fatal("the session did not end")
			}
			var sessionErr error
			select {
			case sessionErr = <-errChan:
			default:
			}
			assert.Equal(t, int32(2), dials.Load(), "expected no handover after the session ended")
			if tt.wantErr == nil {
				assert.NoError(t, sessionErr)
				return
			}
			assert.ErrorIs(t, sessionErr, tt.wantErr)
			assert.NotErrorIs(t, sessionErr, ErrHandoverFailed)
		})
	}
}

// closeSignalingSource reports when the proxy closes it, which happens during the proxy's teardown.
type closeSignalingSource struct {
	io.ReadCloser
	closed chan struct{}
}

func (s *closeSignalingSource) Close() error {
	err := s.ReadCloser.Close()
	close(s.closed)
	return err
}

// The receiving loop also runs on the server. If sshd's stdout ends while the server reads the
// client's normal close acknowledgment for a handover, the server must still finish the swap, so its
// teardown closes the replacement connection gracefully and the client sees a clean exit.
func TestServerEOFDuringHandoverIsACleanExit(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	serverProxy := newProxyConnection(nil)
	serverInput, serverWriter := io.Pipe()
	defer serverWriter.Close()
	source := &closeSignalingSource{ReadCloser: serverInput, closed: make(chan struct{})}
	serverReady := make(chan struct{})
	ackReceived := make(chan struct{})
	releaseAck := make(chan struct{})
	serverDone := make(chan error, 1)
	handoverDone := make(chan error, 1)
	var accepted atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !accepted.CompareAndSwap(false, true) {
			handoverDone <- serverProxy.acceptHandover(ctx, w, r)
			return
		}
		if err := serverProxy.accept(w, r); err != nil {
			serverDone <- err
			return
		}
		// Hold the receiving loop right after it reads the client's close acknowledgment, until
		// the server's source has ended and its teardown has started.
		conn := serverProxy.conn.Load()
		defaultCloseHandler := conn.CloseHandler()
		conn.SetCloseHandler(func(code int, text string) error {
			err := defaultCloseHandler(code, text)
			close(ackReceived)
			select {
			case <-releaseAck:
			case <-ctx.Done():
			}
			return err
		})
		close(serverReady)
		// Mirrors runServerProxy.
		err := serverProxy.start(ctx, source, io.Discard)
		closeProxyConnection(ctx, serverProxy)
		serverDone <- err
	}))
	defer server.Close()

	clientInput, clientWriter := io.Pipe()
	defer clientWriter.Close()
	ticks := make(chan time.Time, 1)
	clientDone := make(chan error, 1)
	go func() {
		clientDone <- RunClientProxy(ctx, clientInput, io.Discard, func() <-chan time.Time { return ticks }, time.Hour, false,
			func(dialCtx context.Context, _ DialRequest) (*websocket.Conn, error) {
				conn, _, err := websocket.DefaultDialer.DialContext(dialCtx, "ws"+server.URL[4:], nil) // nolint:bodyclose
				return conn, err
			})
	}()

	select {
	case <-serverReady:
	case <-ctx.Done():
		t.Fatal("the server did not accept the connection")
	}
	// The client waits for the server's first byte before it treats the session as started.
	_, err := serverWriter.Write([]byte("SSH-2.0-test\r\n"))
	require.NoError(t, err)
	ticks <- time.Now()
	select {
	case <-ackReceived:
	case <-ctx.Done():
		t.Fatal("the server did not receive the handover close acknowledgment")
	}
	require.NoError(t, serverWriter.Close())
	select {
	case <-source.closed:
	case <-ctx.Done():
		t.Fatal("the server did not start its teardown")
	}
	close(releaseAck)

	select {
	case <-handoverDone:
	case <-ctx.Done():
		t.Fatal("the server handover did not finish")
	}
	select {
	case err := <-serverDone:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("the server did not finish")
	}
	select {
	case err := <-clientDone:
		require.NoError(t, err, "a clean server exit during a handover must be a clean client exit")
	case <-ctx.Done():
		t.Fatal("the client did not finish")
	}
}

// TestClientExitsWhenServerCommandFails reproduces the missing-sshd case: the server accepts the
// websocket but can't launch its command, so it closes the connection immediately. The client
// proxy must exit promptly instead of hanging on the handover goroutine (which would leave the
// ssh client waiting until its ConnectTimeout).
func TestClientExitsWhenServerCommandFails(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	connections := NewConnectionsManager(2, time.Hour)
	server := httptest.NewServer(NewProxyServer(ctx, connections, func(ctx context.Context) *exec.Cmd {
		// A binary that does not exist: serverCmd.Start() fails, mirroring a missing /usr/sbin/sshd.
		return exec.CommandContext(ctx, "databricks-ssh-nonexistent-binary")
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	createConn := func(ctx context.Context, dial DialRequest) (*websocket.Conn, error) {
		conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s?id=%s", wsURL, dial.ConnID), nil) // nolint:bodyclose
		return conn, err
	}
	// Source is never closed by the test; only the server-side close must drive the client to exit.
	src, _ := io.Pipe()
	requestHandoverTick := func() <-chan time.Time { return time.After(time.Hour) }

	done := make(chan error, 1)
	go func() {
		done <- RunClientProxy(ctx, src, io.Discard, requestHandoverTick, time.Hour, false, createConn)
	}()

	select {
	case <-done:
		// Returned promptly after the server closed the connection — no hang.
	case <-time.After(10 * time.Second):
		t.Fatal("RunClientProxy hung after the server closed the connection")
	}
}

// TestClientTimesOutWhenServerSendsNothing reproduces the harder missing-sshd case: the server
// accepts the websocket but holds it open without ever sending the SSH banner (the real server
// does this after failing to launch sshd). The client must abort on the handshake timeout rather
// than block forever on its read loops.
func TestClientTimesOutWhenServerSendsNothing(t *testing.T) {
	original := clientHandshakeTimeout
	clientHandshakeTimeout = 300 * time.Millisecond
	defer func() { clientHandshakeTimeout = original }()

	ctx := cmdio.MockDiscard(t.Context())
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// Hold the connection open, sending nothing, until the client goes away.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	createConn := func(ctx context.Context, dial DialRequest) (*websocket.Conn, error) {
		conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s?id=%s", wsURL, dial.ConnID), nil) // nolint:bodyclose
		return conn, err
	}
	src, _ := io.Pipe()
	requestHandoverTick := func() <-chan time.Time { return time.After(time.Hour) }

	done := make(chan error, 1)
	go func() {
		done <- RunClientProxy(ctx, src, io.Discard, requestHandoverTick, time.Hour, false, createConn)
	}()

	select {
	case err := <-done:
		require.ErrorIs(t, err, errHandshakeTimeout)
	case <-time.After(10 * time.Second):
		t.Fatal("RunClientProxy did not abort on the handshake timeout")
	}
}
