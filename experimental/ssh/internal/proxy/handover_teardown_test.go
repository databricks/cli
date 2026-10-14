package proxy

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

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
	clientOutput := newTestBuffer(t)
	ticks := make(chan time.Time, 1)
	clientDone := make(chan error, 1)
	go func() {
		clientDone <- RunClientProxy(ctx, clientInput, clientOutput, func() <-chan time.Time { return ticks }, time.Hour, false,
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
	// The client waits for the server's first byte before it treats the session as started. Wait
	// until the client has it: the write only proves the sending loop read it, and a handover that
	// takes the mutex before the sending loop sends it would block the loop from reading the EOF.
	banner := []byte("SSH-2.0-test\r\n")
	_, err := serverWriter.Write(banner)
	require.NoError(t, err)
	require.NoError(t, clientOutput.WaitForWrite(banner))
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

// If the client's input ends while its receiving loop holds the server's normal handover close, the
// client must still complete the swap: its exit then sends the resumable "finished" close on the
// replacement connection, and the server ends the session at once instead of waiting for a reattach.
func TestClientEOFDuringHandoverReleasesServer(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	serverProxy := newResumableProxyConnection(nil)
	serverInput, serverWriter := io.Pipe()
	defer serverWriter.Close()
	serverReady := make(chan struct{})
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
		close(serverReady)
		// Mirrors runServerProxy.
		err := serverProxy.start(ctx, serverInput, io.Discard)
		closeProxyConnection(ctx, serverProxy)
		serverDone <- err
	}))
	defer server.Close()

	clientInput, clientWriter := io.Pipe()
	defer clientWriter.Close()
	clientSource := &closeSignalingSource{ReadCloser: clientInput, closed: make(chan struct{})}
	clientOutput := newTestBuffer(t)
	ticks := make(chan time.Time, 1)
	clientDone := make(chan error, 1)
	ackSent := make(chan struct{})
	releaseAck := make(chan struct{})
	var dials atomic.Int32
	go func() {
		clientDone <- RunClientProxy(ctx, clientSource, clientOutput, func() <-chan time.Time { return ticks }, time.Hour, true,
			func(dialCtx context.Context, _ DialRequest) (*websocket.Conn, error) {
				conn, _, err := websocket.DefaultDialer.DialContext(dialCtx, "ws"+server.URL[4:], nil) // nolint:bodyclose
				// Hold the client's receiving loop right after it acknowledges the server's
				// handover close on the first connection, until the client's teardown has started.
				if err == nil && dials.Add(1) == 1 {
					defaultCloseHandler := conn.CloseHandler()
					conn.SetCloseHandler(func(code int, text string) error {
						err := defaultCloseHandler(code, text)
						close(ackSent)
						select {
						case <-releaseAck:
						case <-ctx.Done():
						}
						return err
					})
				}
				return conn, err
			})
	}()

	select {
	case <-serverReady:
	case <-ctx.Done():
		t.Fatal("the server did not accept the connection")
	}
	banner := []byte("SSH-2.0-test\r\n")
	_, err := serverWriter.Write(banner)
	require.NoError(t, err)
	require.NoError(t, clientOutput.WaitForWrite(banner))
	ticks <- time.Now()
	select {
	case <-ackSent:
	case <-ctx.Done():
		t.Fatal("the client did not acknowledge the handover close")
	}
	select {
	case err := <-handoverDone:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("the server did not finish the handover")
	}
	require.NoError(t, clientWriter.Close())
	select {
	case <-clientSource.closed:
	case <-ctx.Done():
		t.Fatal("the client did not start its teardown")
	}
	close(releaseAck)

	select {
	case err := <-clientDone:
		require.NoError(t, err)
	case <-ctx.Done():
		t.Fatal("the client did not finish")
	}
	select {
	case err := <-serverDone:
		require.NoError(t, err)
	case <-serverProxy.resume.parked:
		t.Fatalf("a clean client exit left the server waiting %v for a reattach", serverProxy.resume.grace)
	case <-ctx.Done():
		t.Fatal("the server did not finish after a clean client exit")
	}
}
