//go:build !windows

package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The invariant the whole resume protocol exists to hold: a reset must not cost or duplicate a
// single byte. SSH verifies a MAC over the byte stream (RFC 4253 section 6), so a resume that got
// this wrong would disconnect the session rather than repair it - a worse failure than the drop.
//
// The reset lands immediately after a burst of writes, while the echo of that burst is still in
// flight, so the server has bytes it must genuinely replay.
func TestResumeLosesNoBytesWhenResetMidStream(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	relay := newTCPRelay(t, server.Listener.Addr().String())

	errChan := make(chan error, 1)
	client := createResumableTestClient(t, relay.URL(), errChan)
	defer client.Cleanup()

	const lines = 500
	var expected []byte
	for i := range lines {
		line := fmt.Appendf(nil, "line %d\n", i)
		_, err := client.InputWriter.Write(line)
		require.NoError(t, err)
		expected = append(expected, line...)
	}

	relay.resetClients(t)

	require.NoError(t, client.Output.WaitForWrite(fmt.Appendf(nil, "line %d\n", lines-1)),
		"the session did not survive the reset")
	assert.Equal(t, string(expected), client.Output.String())

	select {
	case err := <-errChan:
		t.Fatalf("session ended despite being resumable: %v", err)
	default:
	}
}

// Customers report drops several times a session, so surviving one reset is not enough.
func TestResumeSurvivesRepeatedResets(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	relay := newTCPRelay(t, server.Listener.Addr().String())

	errChan := make(chan error, 1)
	client := createResumableTestClient(t, relay.URL(), errChan)
	defer client.Cleanup()

	const rounds = 4
	const linesPerRound = 100
	var expected []byte
	for round := range rounds {
		for i := range linesPerRound {
			line := fmt.Appendf(nil, "round %d line %d\n", round, i)
			_, err := client.InputWriter.Write(line)
			require.NoError(t, err)
			expected = append(expected, line...)
		}
		relay.resetClients(t)
		last := fmt.Appendf(nil, "round %d line %d\n", round, linesPerRound-1)
		require.NoError(t, client.Output.WaitForWrite(last), "round %d did not survive its reset", round)
	}

	assert.Equal(t, string(expected), client.Output.String())

	select {
	case err := <-errChan:
		t.Fatalf("session ended despite being resumable: %v", err)
	default:
	}
}

// A client that never comes back must not pin sshd and a client slot forever. Releasing the slot
// is also what restarts the shutdown timer, so without this a single dropped session would keep
// the whole server alive until its own timeout.
func TestServerReleasesASessionThatIsNeverReattached(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	connections := NewConnectionsManager(2, time.Hour)
	proxyServer := NewProxyServer(ctx, connections, func(ctx context.Context) *exec.Cmd {
		return exec.CommandContext(ctx, "cat", "-u")
	})
	proxyServer.resumeGrace = 300 * time.Millisecond
	server := httptest.NewServer(proxyServer)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?id=abandoned&resume_version=2&delivered=0", nil) // nolint:bodyclose
	require.NoError(t, err)

	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte("hello\n")))
	_, _, err = conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, 1, connections.Count())

	tcpConn, ok := conn.UnderlyingConn().(*net.TCPConn)
	require.True(t, ok)
	require.NoError(t, tcpConn.SetLinger(0))
	require.NoError(t, tcpConn.Close())

	require.Eventually(t, func() bool {
		return connections.Count() == 0
	}, 10*time.Second, 20*time.Millisecond, "the server held the session past its resume grace period")
}

// A reattach for a session the server no longer holds must be refused. Starting a fresh session
// instead would hand the client a new sshd, and replaying into that fails the SSH stream with a
// corrupted MAC instead of a clear error.
func TestReattachToAnUnknownSessionIsRefused(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	_, resp, err := websocket.DefaultDialer.Dial(wsURL+"?id=never-existed&resume_version=2&delivered=0&reattach=1", nil) // nolint:bodyclose
	require.Error(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusGone, resp.StatusCode)
}

// Drives the wire protocol by hand to pin the reattach exchange itself, with a replay that is
// deliberately non-empty: the client under-reports its delivered offset as zero, so the server has
// to replay the whole session. Asserts the ordering the client depends on - the delivered offset
// arrives as a text frame first, and the replayed payload follows it.
func TestReattachReplaysFromTheOffsetTheClientReports(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	const sessionID = "replay-session"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL+"?id="+sessionID+"&resume_version=2&delivered=0", nil) // nolint:bodyclose
	require.NoError(t, err)

	const payload = "hello\n"
	require.NoError(t, conn.WriteMessage(websocket.BinaryMessage, []byte(payload)))
	mt, echo, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, websocket.BinaryMessage, mt)
	require.Equal(t, payload, string(echo))

	// Kill the connection without a close handshake, so the server treats it as a drop and parks
	// the session instead of tearing it down.
	tcpConn, ok := conn.UnderlyingConn().(*net.TCPConn)
	require.True(t, ok)
	require.NoError(t, tcpConn.SetLinger(0))
	require.NoError(t, tcpConn.Close())

	// Reattach claiming to have delivered nothing, so the replay covers the whole echo.
	resumed, _, err := websocket.DefaultDialer.Dial(wsURL+"?id="+sessionID+"&resume_version=2&delivered=0&reattach=1", nil) // nolint:bodyclose
	require.NoError(t, err)
	defer resumed.Close()

	mt, greeting, err := resumed.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, websocket.TextMessage, mt, "the reattach handshake must arrive before any payload")
	var msg controlMessage
	require.NoError(t, json.Unmarshal(greeting, &msg))
	assert.Equal(t, int64(len(payload)), msg.Delivered, "the server wrote our payload to sshd, so that is its delivered count")

	mt, replayed, err := resumed.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, websocket.BinaryMessage, mt)
	assert.Equal(t, payload, string(replayed), "the server must replay the echo the client claimed not to have")
}

// A reattach without the offset it replays from is meaningless, and silently treating it as a new
// session is the failure this protocol is designed to avoid.
func TestReattachWithoutADeliveredOffsetIsRejected(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	_, resp, err := websocket.DefaultDialer.Dial(wsURL+"?id=some-session&reattach=1", nil) // nolint:bodyclose
	require.Error(t, err)
	require.NotNil(t, resp)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
