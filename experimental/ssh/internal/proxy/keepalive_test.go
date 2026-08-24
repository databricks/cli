package proxy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

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

func keepaliveTestDialer(serverURL string, onConn func(*websocket.Conn)) createWebsocketConnectionFunc {
	wsURL := "ws" + serverURL[4:]
	return func(ctx context.Context, connID string) (*websocket.Conn, error) {
		conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s?id=%s", wsURL, connID), nil) // nolint:bodyclose
		if err != nil {
			return nil, err
		}
		if onConn != nil {
			onConn(conn)
		}
		return conn, nil
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

	failWrites := func(conn *websocket.Conn) {
		// A deadline in the past fails every write on this connection, so every ping fails.
		// Set before the connection is handed to the proxy, so no writer can be in flight.
		// Reads are unaffected: the connection is otherwise healthy and the session must survive.
		conn.SetWriteDeadline(time.Now().Add(-time.Second)) // nolint:errcheck
	}

	src, _ := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- RunClientProxy(ctx, src, io.Discard, neverTick, 20*time.Millisecond, keepaliveTestDialer(server.URL, failWrites))
	}()

	// Long enough for many pings to be attempted and fail.
	select {
	case err := <-done:
		t.Fatalf("session ended after a failed keepalive ping: %v", err)
	case <-time.After(2 * time.Second):
	}
}

// TestKeepalivePingBlockedByHandoverDoesNotDeadlock asserts the invariant the keepalive design
// rests on: a ping sent through the proxy's serialised write path blocks for the duration of a
// handover, and a handover waits on the receiving loop rather than that write path, so the two
// cannot deadlock each other.
func TestKeepalivePingBlockedByHandoverDoesNotDeadlock(t *testing.T) {
	ctx := t.Context()

	server := setupTestServer(ctx, t)
	defer server.Cleanup()

	// Holds the handover open at the point where it has taken the write path but has not yet
	// completed, which is when a ping must block rather than break the handover.
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
		pingDone <- client.Proxy.sendMessage(websocket.PingMessage, nil)
	}()

	select {
	case err := <-pingDone:
		t.Fatalf("ping was written while a handover held the write path: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseHandover)

	select {
	case err := <-handoverDone:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("handover deadlocked while a keepalive ping waited on the write path")
	}
	select {
	case err := <-pingDone:
		require.NoError(t, err)
	case <-time.After(10 * time.Second):
		t.Fatal("keepalive ping never completed after the handover finished")
	}
}
