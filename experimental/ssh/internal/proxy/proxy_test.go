package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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
		if err != nil && !errors.Is(err, context.Canceled) {
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

func setupTestClient(ctx context.Context, t *testing.T, serverURL string) *TestProxy {
	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("Client", true))
	clientInput, clientInputWriter := io.Pipe()
	clientOutput := newTestBuffer(t)
	wsURL := "ws" + serverURL[4:]
	clientProxy := newProxyConnection(func(ctx context.Context, connID string) (*websocket.Conn, error) {
		return createTestWebsocketConnection(wsURL)
	})
	err := clientProxy.connect(ctx)
	require.NoError(t, err)

	wg := sync.WaitGroup{}
	wg.Go(func() {
		err := clientProxy.start(ctx, clientInput, clientOutput)
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("proxy error: %v", err)
		}
	})

	cleanup := func() {
		clientProxy.close()
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
