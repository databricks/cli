package ssh

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testBuffer struct {
	t       *testing.T
	buff    *bytes.Buffer
	OnWrite chan []byte
}

const MAX_BUFFER_SIZE = 1024

func newTestBuffer(t *testing.T) *testBuffer {
	return &testBuffer{
		t:       t,
		buff:    new(bytes.Buffer),
		OnWrite: make(chan []byte, MAX_BUFFER_SIZE),
	}
}

func (tb *testBuffer) String() string {
	return tb.buff.String()
}

func (tb *testBuffer) Read(p []byte) (n int, err error) {
	return tb.buff.Read(p)
}

func (tb *testBuffer) Write(p []byte) (n int, err error) {
	n, err = tb.buff.Write(p)
	require.NoError(tb.t, err)
	tb.OnWrite <- p
	return n, err
}

func (tb *testBuffer) AssertWrite(expected []byte) {
	select {
	case data := <-tb.OnWrite:
		assert.Equal(tb.t, expected, data)
	case <-time.After(2 * time.Second):
		tb.t.Error("timeout waiting for write, was expecting: " + string(expected))
	}
}

type TestProxy struct {
	Proxy       *proxyConnection
	InputWriter io.Writer
	Output      *testBuffer
	URL         string
	Cleanup     func()
}

func setupTestServer(ctx context.Context, t *testing.T) *TestProxy {
	serverInput, serverInputWriter := io.Pipe()
	serverOutput := newTestBuffer(t)
	var serverProxy *proxyConnection
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serverProxy != nil {
			err := serverProxy.AcceptHandover(ctx, w, r)
			if err != nil {
				t.Errorf("failed to accept handover: %v", err)
			}
			return
		}
		serverProxy = newProxyConnection(nil)
		err := serverProxy.Accept(w, r)
		if err != nil {
			t.Errorf("failed to accept websocket connection: %v", err)
			return
		}
		defer serverProxy.Close()
		err = serverProxy.Start(ctx, serverInput, serverOutput)
		if err != nil && !errors.Is(err, errProxyEOF) {
			t.Errorf("server error: %v", err)
			return
		}
	}))
	cleanup := func() {
		server.Close()
		serverProxy.Close()
		serverInputWriter.Close()
	}
	return &TestProxy{
		Proxy:       serverProxy,
		InputWriter: serverInputWriter,
		Output:      serverOutput,
		Cleanup:     cleanup,
		URL:         server.URL,
	}
}

func createTestWebsocketConnection(url string) (*websocket.Conn, error) {
	conn, _, err := websocket.DefaultDialer.Dial(url, nil) // nolint:bodyclose
	return conn, err
}

func setupTestClient(ctx context.Context, t *testing.T, serverURL string) *TestProxy {
	clientInput, clientInputWriter := io.Pipe()
	clientOutput := newTestBuffer(t)
	wsURL := "ws" + serverURL[4:]
	clientProxy := newProxyConnection(func(ctx context.Context, connID string) (*websocket.Conn, error) {
		return createTestWebsocketConnection(wsURL)
	})
	err := clientProxy.Connect(ctx)
	require.NoError(t, err)

	go func() {
		err := clientProxy.Start(ctx, clientInput, clientOutput)
		if err != nil && !errors.Is(err, errProxyEOF) {
			t.Errorf("proxy error: %v", err)
		}
	}()

	cleanup := func() {
		clientProxy.Close()
		clientInputWriter.Close()
	}

	return &TestProxy{
		Proxy:       clientProxy,
		InputWriter: clientInputWriter,
		Output:      clientOutput,
		Cleanup:     cleanup,
		URL:         wsURL,
	}
}

func TestClientServerExchange(t *testing.T) {
	ctx := t.Context()

	server := setupTestServer(ctx, t)
	defer server.Cleanup()

	client := setupTestClient(ctx, t, server.URL)
	defer client.Cleanup()

	_, err := client.InputWriter.Write([]byte("Hello from client"))
	require.NoError(t, err)
	server.Output.AssertWrite([]byte("Hello from client"))

	_, err = server.InputWriter.Write([]byte("Hello from server"))
	require.NoError(t, err)
	client.Output.AssertWrite([]byte("Hello from server"))

	_, err = client.InputWriter.Write([]byte("Hello again from client"))
	require.NoError(t, err)
	_, err = server.InputWriter.Write([]byte("Hello again from server"))
	require.NoError(t, err)
	client.Output.AssertWrite([]byte("Hello again from server"))
	server.Output.AssertWrite([]byte("Hello again from client"))

	assert.Equal(t, "Hello from clientHello again from client", server.Output.String())
	assert.Equal(t, "Hello from serverHello again from server", client.Output.String())
}

func TestConnectionHandover(t *testing.T) {
	ctx := t.Context()

	server := setupTestServer(ctx, t)
	defer server.Cleanup()

	client := setupTestClient(ctx, t, server.URL)
	initialProxyConn := client.Proxy.conn.Load().(*websocket.Conn)
	defer client.Cleanup()

	const numMessages = MAX_BUFFER_SIZE - 1
	clientMessages := make([]string, numMessages)
	serverMessages := make([]string, numMessages)
	for i := range numMessages {
		clientMessages[i] = fmt.Sprintf("client-msg-%d", i)
		serverMessages[i] = fmt.Sprintf("server-msg-%d", i)
	}

	handoverChan := make(chan struct{})

	go func() {
		for i := range numMessages {
			client.InputWriter.Write([]byte(clientMessages[i])) // nolint:errcheck
			server.InputWriter.Write([]byte(serverMessages[i])) // nolint:errcheck
			if i == numMessages/2 {
				t.Logf("Triggering handover on message number %d", i)
				handoverChan <- struct{}{}
			}
		}
	}()

	go func() {
		<-handoverChan
		err := client.Proxy.InitiateHandover(ctx)
		if err != nil {
			t.Errorf("failed to initiate handover: %v", err)
		}
	}()

	for i := range numMessages {
		server.Output.AssertWrite([]byte(clientMessages[i]))
		client.Output.AssertWrite([]byte(serverMessages[i]))
	}
	assert.NotEqual(t, initialProxyConn, client.Proxy.conn.Load())
}
