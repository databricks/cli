//go:build !windows

package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tcpRelay stands in for the workspace front door: it forwards TCP between the client and the
// SSH proxy server, and can reset the client leg the way a load balancer recycling a target
// does. httptest's own CloseClientConnections is no use here - it does not touch the hijacked
// connections a websocket upgrade leaves behind.
type tcpRelay struct {
	listener    net.Listener
	upstream    string
	mu          sync.Mutex
	clientConns []*net.TCPConn
}

func newTCPRelay(t *testing.T, upstream string) *tcpRelay {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	relay := &tcpRelay{listener: listener, upstream: upstream}
	t.Cleanup(func() { listener.Close() })
	go relay.serve()
	return relay
}

func (r *tcpRelay) serve() {
	for {
		downstream, err := r.listener.Accept()
		if err != nil {
			return
		}
		upstream, err := net.Dial("tcp", r.upstream)
		if err != nil {
			downstream.Close()
			return
		}
		r.mu.Lock()
		r.clientConns = append(r.clientConns, downstream.(*net.TCPConn))
		r.mu.Unlock()
		// Both directions end with the connection being torn down, so a copy error is the
		// expected way for these to finish.
		go func() {
			_, _ = io.Copy(upstream, downstream)
			upstream.Close()
		}()
		go func() {
			_, _ = io.Copy(downstream, upstream)
			downstream.Close()
		}()
	}
}

// resetClients sends a TCP RST on every client leg, so the client's next read fails with
// "connection reset by peer" rather than seeing a clean close. SetLinger is asserted: without
// it the close is graceful and the test would exercise the wrong path. Connections are taken off
// the list as they are reset, so a later call only touches legs opened since.
func (r *tcpRelay) resetClients(t *testing.T) {
	r.mu.Lock()
	conns := r.clientConns
	r.clientConns = nil
	r.mu.Unlock()
	for _, conn := range conns {
		require.NoError(t, conn.SetLinger(0))
		require.NoError(t, conn.Close())
	}
}

// createResumableTestClient builds a client with the resume protocol enabled, dialing through a
// URL that mirrors the production one: the delivered offset rides along on every dial, and a
// reattach says so explicitly.
func createResumableTestClient(t *testing.T, serverURL string, errChan chan error) *testClient {
	wsURL := "ws" + serverURL[4:]
	createConn := func(ctx context.Context, dial DialRequest) (*websocket.Conn, error) {
		url := fmt.Sprintf("%s?id=%s", wsURL, dial.ConnID)
		if dial.ResumeCapable {
			url += fmt.Sprintf("&delivered=%d", dial.Delivered)
			if dial.Reattach {
				url += "&reattach=1"
			}
		}
		conn, _, err := websocket.DefaultDialer.Dial(url, nil) // nolint:bodyclose
		return conn, err
	}
	return createTestClientWithDialer(t, createConn, nil, time.Hour, true, errChan)
}

// URL returns the relay's address in the http form createTestClient expects.
func (r *tcpRelay) URL() string {
	return "http://" + r.listener.Addr().String()
}

// A mid-session reset on a connection without resume negotiated must end the session and surface
// as ErrWebsocketDropped. Both halves matter: telemetry needs to tell a dropped session apart from
// a clean exit, and this is also the fallback a client talking to an older server relies on, so it
// has to keep working unchanged.
func TestMidSessionResetIsAttributedAsADrop(t *testing.T) {
	server := createTestServer(t, 2, time.Hour)
	defer server.Close()

	relay := newTCPRelay(t, server.Listener.Addr().String())

	errChan := make(chan error, 1)
	client := createTestClient(t, relay.URL(), nil, time.Hour, errChan)
	defer client.Cleanup()

	msg := []byte("before drop\n")
	_, err := client.InputWriter.Write(msg)
	require.NoError(t, err)
	require.NoError(t, client.Output.AssertWrite(msg))

	relay.resetClients(t)

	select {
	case err := <-errChan:
		assert.ErrorIs(t, err, ErrWebsocketDropped)
	case <-time.After(10 * time.Second):
		t.Fatal("the client proxy did not report the dropped connection")
	}
}

// The error that ends a session must survive the cancellation it triggers. proxy.start cancels the
// context before errgroup records its error, so the handover and keepalive goroutines wake up and
// return first; if they reported the cancellation as their own error, errgroup would keep that one
// and normalizeProxyError would turn a dropped session into a clean exit. The window is small, so
// this runs the drop repeatedly rather than once.
func TestADropIsNeverReportedAsACleanExit(t *testing.T) {
	for attempt := range 25 {
		server := createTestServer(t, 2, time.Hour)
		relay := newTCPRelay(t, server.Listener.Addr().String())

		errChan := make(chan error, 1)
		client := createTestClient(t, relay.URL(), nil, time.Hour, errChan)

		msg := []byte("before drop\n")
		_, err := client.InputWriter.Write(msg)
		require.NoError(t, err)
		require.NoError(t, client.Output.AssertWrite(msg))

		relay.resetClients(t)

		select {
		case err := <-errChan:
			require.ErrorIs(t, err, ErrWebsocketDropped, "attempt %d", attempt)
		case <-time.After(10 * time.Second):
			t.Fatalf("attempt %d: the drop was reported as a clean exit", attempt)
		}
		client.Cleanup()
		server.Close()
	}
}
