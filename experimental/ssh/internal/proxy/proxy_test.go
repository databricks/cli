package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testBuffer struct {
	t       *testing.T
	m       sync.Mutex
	buff    *bytes.Buffer
	OnWrite chan []byte
}

const (
	MESSAGE_CHUNKS      = 4
	MESSAGES_PER_CHUNK  = 512
	TOTAL_MESSAGE_COUNT = MESSAGE_CHUNKS * MESSAGES_PER_CHUNK
)

func newTestBuffer(t *testing.T) *testBuffer {
	return &testBuffer{
		t:       t,
		m:       sync.Mutex{},
		buff:    new(bytes.Buffer),
		OnWrite: make(chan []byte, TOTAL_MESSAGE_COUNT),
	}
}

func (tb *testBuffer) String() string {
	return tb.buff.String()
}

func (tb *testBuffer) Read(p []byte) (n int, err error) {
	return tb.buff.Read(p)
}

func (tb *testBuffer) Write(p []byte) (n int, err error) {
	tb.m.Lock()
	n, err = tb.buff.Write(p)
	tb.m.Unlock()
	require.NoError(tb.t, err)
	tb.OnWrite <- p
	return n, err
}

func (tb *testBuffer) AssertWrite(expected []byte) error {
	select {
	case data := <-tb.OnWrite:
		assert.Equal(tb.t, expected, data)
		return nil
	case <-time.After(3 * time.Second):
		return errors.New("timeout waiting for write, was expecting: " + string(expected))
	}
}

func (tb *testBuffer) Contains(data []byte) bool {
	tb.m.Lock()
	defer tb.m.Unlock()
	return bytes.Contains(tb.buff.Bytes(), data)
}

func (tb *testBuffer) WaitForWrite(expected []byte) error {
	for {
		select {
		case <-tb.OnWrite:
			if tb.Contains(expected) {
				return nil
			}
		case <-time.After(3 * time.Second):
			return errors.New("timeout waiting for write")
		}
	}
}

type TestProxy struct {
	Proxy   *proxyConnection
	Input   io.Writer
	Output  *testBuffer
	URL     string
	Cleanup func()
}

func setupTestServer(ctx context.Context, t *testing.T) *TestProxy {
	serverInput, serverInputWriter := io.Pipe()
	serverOutput := newTestBuffer(t)
	var serverProxy *proxyConnection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := log.NewContext(ctx, log.GetLogger(ctx).With("Server", true))
		if serverProxy != nil {
			err := serverProxy.acceptHandover(ctx, w, r)
			if err != nil {
				t.Errorf("failed to accept handover: %v", err)
			}
			return
		}
		serverProxy = newProxyConnection(nil)
		err := serverProxy.accept(w, r)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}
		defer serverProxy.close()
		err = serverProxy.start(ctx, serverInput, serverOutput)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, io.ErrClosedPipe) {
			t.Errorf("server error: %v", err)
			return
		}
	}))
	cleanup := func() {
		server.Close()
		serverProxy.close()
		serverInputWriter.Close()
	}
	return &TestProxy{
		Proxy:   serverProxy,
		Input:   serverInputWriter,
		Output:  serverOutput,
		Cleanup: cleanup,
		URL:     server.URL,
	}
}

func createTestWebsocketConnection(url string) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil) // nolint:bodyclose
	return conn, err
}

// neverTick is a tick channel that never fires, for tests that don't exercise a periodic behaviour.
func neverTick() <-chan time.Time {
	return time.After(time.Hour)
}

func setupTestClient(ctx context.Context, t *testing.T, serverURL string) *TestProxy {
	return setupTestClientWithDialHook(ctx, t, serverURL, nil)
}

// setupTestClientWithDialHook is setupTestClient with a hook called on every websocket dial: the
// initial connection and each one a handover creates.
func setupTestClientWithDialHook(ctx context.Context, t *testing.T, serverURL string, onDial func()) *TestProxy {
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("Client", true))
	clientInput, clientInputWriter := io.Pipe()
	clientOutput := newTestBuffer(t)
	wsURL := "ws" + serverURL[4:]
	clientProxy := newProxyConnection(func(ctx context.Context, dial DialRequest) (*websocket.Conn, error) {
		if onDial != nil {
			onDial()
		}
		return createTestWebsocketConnection(wsURL)
	})
	err := clientProxy.connect(ctx)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Go(func() {
		err := clientProxy.start(ctx, clientInput, clientOutput)
		if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, io.ErrClosedPipe) {
			t.Errorf("proxy error: %v", err)
		}
	})

	cleanup := func() {
		clientProxy.close()
		clientInput.Close()
		clientInputWriter.Close()
		wg.Wait()
	}

	return &TestProxy{
		Proxy:   clientProxy,
		Input:   clientInputWriter,
		Output:  clientOutput,
		Cleanup: cleanup,
		URL:     wsURL,
	}
}

func TestClientServerExchange(t *testing.T) {
	ctx := t.Context()

	server := setupTestServer(ctx, t)
	defer server.Cleanup()

	client := setupTestClient(ctx, t, server.URL)
	defer client.Cleanup()

	_, err := client.Input.Write(createTestMessage("client", 1))
	require.NoError(t, err)
	err = server.Output.AssertWrite(createTestMessage("client", 1))
	require.NoError(t, err)

	_, err = server.Input.Write(createTestMessage("server", 1))
	require.NoError(t, err)
	err = client.Output.AssertWrite(createTestMessage("server", 1))
	require.NoError(t, err)

	_, err = client.Input.Write(createTestMessage("client", 2))
	require.NoError(t, err)
	_, err = server.Input.Write(createTestMessage("server", 2))
	require.NoError(t, err)
	err = client.Output.AssertWrite(createTestMessage("server", 2))
	require.NoError(t, err)
	err = server.Output.AssertWrite(createTestMessage("client", 2))
	require.NoError(t, err)

	expectedClientOutput := fmt.Sprintf("%s%s", createTestMessage("client", 1), createTestMessage("client", 2))
	expectedServerOutput := fmt.Sprintf("%s%s", createTestMessage("server", 1), createTestMessage("server", 2))
	assert.Equal(t, expectedClientOutput, server.Output.String())
	assert.Equal(t, expectedServerOutput, client.Output.String())
}

func createTestMessage(location string, seq int) []byte {
	return fmt.Appendf(nil, "%s-msg-%d", location, seq)
}

func TestConnectionHandover(t *testing.T) {
	ctx := t.Context()

	server := setupTestServer(ctx, t)
	defer server.Cleanup()

	client := setupTestClient(ctx, t, server.URL)
	defer client.Cleanup()

	handoverChan := make(chan struct{})

	go func() {
		for i := range TOTAL_MESSAGE_COUNT {
			client.Input.Write(createTestMessage("client", i)) // nolint:errcheck
			server.Input.Write(createTestMessage("server", i)) // nolint:errcheck
			if i > 0 && i%MESSAGES_PER_CHUNK == 0 && i < TOTAL_MESSAGE_COUNT-1 {
				handoverChan <- struct{}{}
			}
		}
	}()

	go func() {
		for {
			select {
			case <-handoverChan:
				err := client.Proxy.initiateHandover(ctx)
				if err != nil {
					t.Errorf("failed to initiate handover: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	for i := range TOTAL_MESSAGE_COUNT {
		err := server.Output.AssertWrite(createTestMessage("client", i))
		require.NoError(t, err)
		err = client.Output.AssertWrite(createTestMessage("server", i))
		require.NoError(t, err)
	}
}

// A failed acknowledgement write on a resumable connection must be treated exactly like a failed
// binary write: close the connection and report errSendFailedResumable, so the receiving loop's
// next read fails and drives the reattach. When traffic is one-way from the server the receiving
// side never writes a binary frame, so a poisoned connection surfaces only as a failed ack; left
// as a bare log it would let the peer's replay buffer fill and end the session instead.
func TestFailedAckWriteClosesResumableConnectionToDriveReattach(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// A live peer to dial; drain until the client goes away.
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	conn, err := createTestWebsocketConnection(wsURL)
	require.NoError(t, err)

	pc := newResumableProxyConnection(nil)
	pc.conn.Store(conn)

	// Poison the write side the way gorilla latches it after any failed write, without disturbing
	// reads - the one-way-from-server case where only the ack ever fails.
	require.NoError(t, conn.SetWriteDeadline(time.Now().Add(-time.Hour)))

	// sendControlMessage is the ack path (a text control frame), not a binary payload.
	err = pc.sendControlMessage(42)
	require.ErrorIs(t, err, errSendFailedResumable, "a failed ack write on a resumable connection must report the resumable-send failure that drives the reattach")

	// It must have closed the connection: a second close returns net.ErrClosed. With the
	// connection closed, the receiving loop's next read fails and reattaches, rather than the
	// failure being silently swallowed.
	require.ErrorIs(t, conn.Close(), net.ErrClosed, "sendMessage must close the poisoned connection so the receiving loop reattaches")
}
