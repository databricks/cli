package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

// Write modes for pausableConn.
const (
	connWriteOK = iota
	connWriteFail
	connWritePark
)

var errTestWriteFailed = errors.New("test: socket write failed")

// pausableConn emulates the socket conditions a keepalive meets on a stalled peer: writes can be
// made to fail outright, or to park the way a full send buffer does — blocking until the write
// deadline expires. Wrapping the socket rather than the websocket keeps the production write path
// (gorilla's own locking and deadline handling) in the test.
type pausableConn struct {
	net.Conn
	mode         atomic.Int32
	deadline     atomic.Pointer[time.Time]
	parked       chan struct{}
	signalParked func()
}

func newPausableConn(conn net.Conn) *pausableConn {
	parked := make(chan struct{})
	return &pausableConn{
		Conn:         conn,
		parked:       parked,
		signalParked: sync.OnceFunc(func() { close(parked) }),
	}
}

func (c *pausableConn) SetWriteDeadline(t time.Time) error {
	c.deadline.Store(&t)
	return c.Conn.SetWriteDeadline(t)
}

func (c *pausableConn) Write(p []byte) (int, error) {
	switch c.mode.Load() {
	case connWriteFail:
		return 0, errTestWriteFailed
	case connWritePark:
		c.signalParked()
		if d := c.deadline.Load(); d != nil && !d.IsZero() {
			time.Sleep(time.Until(*d))
		}
		return 0, os.ErrDeadlineExceeded
	}
	return c.Conn.Write(p)
}

// startKeepaliveTestServer stands up a websocket peer that behaves like the SSH server side of the
// tunnel for an idle session: it sends the first bytes (which the client waits for before it
// considers the session established), then only reads. Pings it receives are reported on the
// returned channel and answered with a pong, the same way gorilla's default ping handler does.
func startKeepaliveTestServer(t *testing.T) (*httptest.Server, <-chan struct{}) {
	pings := make(chan struct{}, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetPingHandler(func(appData string) error {
			select {
			case pings <- struct{}{}:
			default:
			}
			return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
		})
		if err := conn.WriteMessage(websocket.BinaryMessage, []byte("SSH-2.0-test\r\n")); err != nil {
			return
		}
		// Ping handlers only run while a read is in progress, so keep reading until the client goes away.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	return server, pings
}

// keepaliveTestDialer returns a connection factory for RunClientProxy. onNetConn, when set, receives
// each connection's underlying socket so a test can control how its writes behave.
func keepaliveTestDialer(serverURL string, onNetConn func(*pausableConn)) createWebsocketConnectionFunc {
	wsURL := "ws" + serverURL[4:]
	dialer := websocket.Dialer{
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := net.Dial(network, addr)
			if err != nil {
				return nil, err
			}
			wrapped := newPausableConn(conn)
			if onNetConn != nil {
				onNetConn(wrapped)
			}
			return wrapped, nil
		},
	}
	return func(ctx context.Context, connID string) (*websocket.Conn, error) {
		conn, _, err := dialer.DialContext(ctx, fmt.Sprintf("%s?id=%s", wsURL, connID), nil) // nolint:bodyclose
		return conn, err
	}
}

// TestKeepalivePingReachesServer covers the fix itself: an idle session sends no data, so without
// the keepalive nothing at all crosses the websocket and the server side eventually reaps the
// stream it considers dead.
func TestKeepalivePingReachesServer(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	server, pings := startKeepaliveTestServer(t)
	defer server.Close()

	// Never written to: the session stays idle for the whole test.
	src, _ := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- RunClientProxy(ctx, src, io.Discard, neverTick, 20*time.Millisecond, keepaliveTestDialer(server.URL, nil))
	}()

	select {
	case <-pings:
	case err := <-done:
		t.Fatalf("session ended before a keepalive ping arrived: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("no keepalive ping arrived at the server")
	}
}

// TestKeepalivePingFailureDoesNotEndSession asserts a keepalive can never be the thing that ends a
// session: it has no information the data loops lack, and it runs in the errgroup that drives them,
// so a returned error would tear the session down.
func TestKeepalivePingFailureDoesNotEndSession(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	server, _ := startKeepaliveTestServer(t)
	defer server.Close()

	var socket atomic.Pointer[pausableConn]
	src, _ := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- RunClientProxy(ctx, src, io.Discard, neverTick, 20*time.Millisecond,
			keepaliveTestDialer(server.URL, func(c *pausableConn) { socket.Store(c) }))
	}()

	// Fail every write once the connection is up, leaving reads healthy: the connection is otherwise
	// fine and the session must survive the pings that then fail.
	require.Eventually(t, func() bool { return socket.Load() != nil }, 10*time.Second, 10*time.Millisecond)
	socket.Load().mode.Store(connWriteFail)

	// Long enough for many pings to be attempted and fail.
	select {
	case err := <-done:
		t.Fatalf("session ended after a failed keepalive ping: %v", err)
	case <-time.After(2 * time.Second):
	}
}

// TestKeepalivePingParkedInWriteDoesNotStallClose covers the hazard of writing from a third
// goroutine: a ping on a stalled or half-open connection parks in the socket write while holding the
// websocket's write lock, which the closing handshake also needs. Its deadline is what keeps that
// from lasting until the kernel abandons its retransmits, minutes later.
//
// Scoped to the ping's own contribution. pausableConn parks only for as long as the write's deadline,
// and a write whose caller set none fails at once, so the pre-existing unbounded park on the data
// path is out of the picture. The session's own shutdown cannot be measured here either: it waits on
// the receiving loop, which unblocks only when the peer reacts to the close frame, and a peer that
// has silently gone away never does — with or without a keepalive.
func TestKeepalivePingParkedInWriteDoesNotStallClose(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	server, _ := startKeepaliveTestServer(t)
	defer server.Close()

	var socket atomic.Pointer[pausableConn]
	proxy := newProxyConnection(keepaliveTestDialer(server.URL, func(c *pausableConn) { socket.Store(c) }))
	require.NoError(t, proxy.connect(ctx))
	require.NotNil(t, socket.Load())

	socket.Load().mode.Store(connWritePark)
	pingDone := make(chan error, 1)
	go func() {
		pingDone <- proxy.sendPing()
	}()
	select {
	case <-socket.Load().parked:
	case <-time.After(10 * time.Second):
		t.Fatal("the keepalive ping did not park in the socket write")
	}

	start := time.Now()
	closeErr := proxy.close()
	require.Less(t, time.Since(start), proxyPingWriteTimeout+5*time.Second,
		"the closing handshake waited on the parked keepalive ping for longer than its write deadline allows")
	// The write itself fails, which close() reports; the point is that it was not held indefinitely.
	require.Error(t, closeErr)

	select {
	case err := <-pingDone:
		require.Error(t, err, "a parked ping write must end in an error, not succeed")
	case <-time.After(10 * time.Second):
		t.Fatal("the keepalive ping never returned from its parked write")
	}
}

// TestKeepalivePingDuringHandoverDoesNotDisruptIt asserts the two periodic behaviours of the tunnel
// stay independent: a ping sent while a handover is in flight neither waits for the handover nor
// breaks it. The keepalive writes from a third goroutine, so nothing else guarantees this.
func TestKeepalivePingDuringHandoverDoesNotDisruptIt(t *testing.T) {
	ctx := t.Context()

	server := setupTestServer(ctx, t)
	defer server.Cleanup()

	// Holds the handover open at the point where it has taken the handover mutex but has not yet
	// swapped the connection, which is when a ping must neither block nor interfere.
	handoverDialing := make(chan struct{})
	releaseHandover := make(chan struct{})
	var dials atomic.Int32

	client := setupTestClientWithDialHook(ctx, t, server.URL, func() {
		if dials.Add(1) > 1 {
			close(handoverDialing)
			<-releaseHandover
		}
	})
	defer client.Cleanup()

	handoverDone := make(chan error, 1)
	go func() {
		handoverDone <- client.Proxy.initiateHandover(ctx)
	}()
	<-handoverDialing

	pingDone := make(chan error, 1)
	go func() {
		pingDone <- client.Proxy.sendPing()
	}()

	select {
	case err := <-pingDone:
		require.NoError(t, err)
	case <-time.After(proxyPingWriteTimeout):
		t.Fatal("keepalive ping waited on the in-flight handover")
	}

	close(releaseHandover)

	select {
	case err := <-handoverDone:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("handover did not complete after a concurrent keepalive ping")
	}

	// The tunnel still carries data on the connection the handover installed.
	_, err := client.Input.Write(createTestMessage("client", 1))
	require.NoError(t, err)
	require.NoError(t, server.Output.WaitForWrite(createTestMessage("client", 1)))
}
