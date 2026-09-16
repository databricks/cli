//go:build !windows

package proxy

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// blackHoleServer accepts a resume-capable websocket, drains every frame the client sends and never
// acknowledges any of it. That is what a peer whose own receiving loop is wedged looks like from
// here: the socket is alive and writes succeed, but its delivered offset never moves.
func blackHoleServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
}

// dropAtOnceServer accepts the upgrade and immediately resets the connection, so the session fails
// before the SSH server's first byte could ever arrive.
func dropAtOnceServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		if tcpConn, ok := conn.UnderlyingConn().(*net.TCPConn); ok {
			_ = tcpConn.SetLinger(0)
		}
		conn.Close()
	}))
}

// serverWithConnections is createTestServer with the connections manager exposed, so a test can
// assert the server released the session, and with the command it runs supplied by the caller.
func serverWithConnections(t *testing.T, resumeGrace time.Duration, command createServerCommandFunc) (*httptest.Server, *ConnectionsManager) {
	ctx := cmdio.MockDiscard(t.Context())
	connections := NewConnectionsManager(1, time.Hour)
	proxyServer := NewProxyServer(ctx, connections, command)
	proxyServer.resumeGrace = resumeGrace
	return httptest.NewServer(proxyServer), connections
}

// dropWebsocket ends a websocket the way a reset does, with no close handshake, so the peer treats
// it as a drop rather than an orderly shutdown.
func dropWebsocket(t *testing.T, conn *websocket.Conn) {
	tcpConn, ok := conn.UnderlyingConn().(*net.TCPConn)
	require.True(t, ok)
	require.NoError(t, tcpConn.SetLinger(0))
	require.NoError(t, tcpConn.Close())
}

// EOF on the source is how every clean exit begins: ssh closes the proxy command's stdin. Waiting
// for the peer to acknowledge the tail is right, but it has to be bounded - a peer that reads
// without ever acknowledging would otherwise keep the CLI alive after the ssh client that spawned
// it is gone.
func TestEOFEndsTheSessionWhenThePeerStopsAcknowledging(t *testing.T) {
	restore := proxyEOFDrainTimeout
	proxyEOFDrainTimeout = 500 * time.Millisecond
	t.Cleanup(func() { proxyEOFDrainTimeout = restore })

	server := blackHoleServer(t)
	defer server.Close()

	srcWriter, done := runResumableClientTo(t, server.URL, &sink{})
	_, err := srcWriter.Write([]byte("hello\n"))
	require.NoError(t, err)
	// Let the write reach the peer, so the bytes are unacknowledged rather than unsent.
	time.Sleep(200 * time.Millisecond)
	require.NoError(t, srcWriter.Close())

	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the client never exited after EOF: it is waiting for an acknowledgment that will never arrive")
	}
}

// The same wait on the server side, where it costs more: the session holds a client slot and keeps
// the shutdown timer cancelled, so a session that is never released keeps the compute alive with
// nobody attached to it.
func TestServerReleasesASessionWhenTheClientStopsAcknowledging(t *testing.T) {
	restore := proxyEOFDrainTimeout
	proxyEOFDrainTimeout = 500 * time.Millisecond
	t.Cleanup(func() { proxyEOFDrainTimeout = restore })

	// sshd's stand-in prints and exits, so the server's sending loop reaches EOF straight away.
	server, connections := serverWithConnections(t, 500*time.Millisecond, func(ctx context.Context) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "echo hello; exit 0")
	})
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?id=no-acks&resume_version=2&delivered=0", nil) // nolint:bodyclose
	require.NoError(t, err)
	defer conn.Close()

	// Read everything so the server's writes never block, but acknowledge nothing.
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	require.Eventually(t, func() bool {
		return connections.Count() == 0
	}, 30*time.Second, 50*time.Millisecond,
		"the server never released the session: its sending loop is still waiting for an acknowledgment of the bytes sshd printed before it exited")
}

// A session that dies before the SSH server's first byte must still report why. That error is what
// the user sees instead of a bare ssh failure, and what telemetry attributes the drop from, so
// losing it turns a failed session into an apparently clean exit.
func TestASessionThatFailsBeforeTheFirstByteStillReportsTheFailure(t *testing.T) {
	server := dropAtOnceServer(t)
	defer server.Close()

	const attempts = 20
	silent := 0
	for range attempts {
		srcWriter, done := runPlainClientTo(t, server.URL, &sink{})
		select {
		case err := <-done:
			if err == nil {
				silent++
			}
		case <-time.After(30 * time.Second):
			t.Fatal("the session neither ended nor reported a failure")
		}
		srcWriter.Close()
	}
	assert.Equal(t, 0, silent,
		"%d of %d sessions that died before the first byte returned no error at all", silent, attempts)
}

// What a user is told when the tunnel connects and then keeps dropping before sshd answers. The
// handshake timeout's own message blames a missing openssh-server, which for a network failure
// sends them to the wrong place - and bills it to the wrong telemetry category.
func TestEarlyDropIsReportedAsADropNotAMissingSshd(t *testing.T) {
	server := dropAtOnceServer(t)
	defer server.Close()

	srcWriter, done := runResumableClientTo(t, server.URL, &sink{})
	defer srcWriter.Close()

	select {
	case err := <-done:
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrWebsocketDropped)
		assert.NotErrorIs(t, err, errHandshakeTimeout)
	case <-time.After(3 * time.Minute):
		t.Fatal("the client never gave up on a connection that drops on every attempt")
	}
}

// A dropped session holds its client slot for the whole grace period, and a user whose network
// flapped long enough for ssh to give up comes back as a new session. This pins the cost: with the
// slots full, that new session is refused until the abandoned ones expire.
func TestAbandonedSessionsHoldTheirClientSlots(t *testing.T) {
	server, _ := serverWithConnections(t, 5*time.Second, func(ctx context.Context) *exec.Cmd {
		return exec.CommandContext(ctx, "cat", "-u")
	})
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?id=first&resume_version=2&delivered=0", nil) // nolint:bodyclose
	require.NoError(t, err)
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("hello\n")))
	_, _, err = conn.ReadMessage()
	require.NoError(t, err)
	dropWebsocket(t, conn)

	_, resp, err := websocket.DefaultDialer.Dial(wsURL+"?id=second&resume_version=2&delivered=0", nil) // nolint:bodyclose
	if resp != nil {
		defer resp.Body.Close()
	}
	require.Error(t, err, "the abandoned session must still hold the only slot")
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
}

// closingServer answers the first dial, sends one payload byte so the client is past its handshake
// window, then ends the session with a normal-closure close frame carrying the given reason. A
// later reattach is refused the way a server that has torn the session down refuses one.
func closingServer(t *testing.T, reason string) (*httptest.Server, *atomic.Int64) {
	var reattachAttempts atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("reattach") == "1" {
			reattachAttempts.Add(1)
			http.Error(w, "Session no longer exists", http.StatusGone)
			return
		}
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		// A handler cannot fail the test directly; a write that fails here shows up as the client
		// never seeing the close frame, which its own assertions report.
		if err := conn.WriteMessage(websocket.BinaryMessage, []byte("x")); err != nil {
			return
		}
		if err := conn.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, reason)); err != nil {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}))
	return server, &reattachAttempts
}

// On a resumable connection the "finished" reason in a close frame is the only thing separating a
// session that ended from a socket that died: a genuine drop arrives as a read error, never as a
// close frame, so the code cannot tell them apart by anything else. That makes every clean exit
// depend on the reason text surviving the path between the client and the workspace. Both readings
// are pinned here, so a change to either side of that contract fails loudly rather than turning
// every clean exit into a reported drop.
func TestNormalClosureReasonDecidesCleanExitVersusDrop(t *testing.T) {
	t.Run("the finished reason ends the session cleanly", func(t *testing.T) {
		server, reattachAttempts := closingServer(t, proxySessionFinished)
		defer server.Close()

		srcWriter, done := runResumableClientTo(t, server.URL, &sink{})
		defer srcWriter.Close()
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(30 * time.Second):
			t.Fatal("the client did not notice the session had finished")
		}
		assert.Zero(t, reattachAttempts.Load(), "a finished session must not be reattached to")
	})

	t.Run("without the reason the same close frame is a drop", func(t *testing.T) {
		server, reattachAttempts := closingServer(t, "")
		defer server.Close()

		srcWriter, done := runResumableClientTo(t, server.URL, &sink{})
		defer srcWriter.Close()
		select {
		case err := <-done:
			assert.ErrorIs(t, err, ErrWebsocketDropped)
		case <-time.After(30 * time.Second):
			t.Fatal("the client hung on a normal closure")
		}
		assert.Equal(t, int64(1), reattachAttempts.Load(),
			"the client tries to reattach, so a stripped reason would warn the user and count a dropped tunnel on every clean exit")
	})
}
