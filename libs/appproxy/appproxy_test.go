package appproxy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/appproxy/testutil"
	"github.com/stretchr/testify/require"

	"github.com/gorilla/websocket"
)

const (
	PROXY_PORT = 0
)

func sendTestRequest(t *testing.T, url, path string) (int, []byte) {
	req, err := http.NewRequest("GET", url+path, bytes.NewBufferString("{'test': 'value'}"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, body
}

func startProxy(t *testing.T, serverAddr string) (*Proxy, string) {
	proxy, err := New(context.Background(), "http://"+serverAddr)
	require.NoError(t, err)

	ln, err := proxy.listen(fmt.Sprintf("localhost:%d", PROXY_PORT))
	require.NoError(t, err)

	go func() {
		_ = proxy.serve(ln)
	}()

	return proxy, ln.Addr().String()
}

func TestProxyStart(t *testing.T) {
	server := testutil.NewHttpServer(t, map[string]string{
		"Content-Type":   "application/json",
		"X-Test-Header":  "test",
		"X-Test-Header2": "test2",
	})

	go func() {
		server.Start()
	}()

	serverAddr := server.Listener.Addr().String()
	proxy, addr := startProxy(t, serverAddr)
	hostUrl := "http://" + addr
	defer func() {
		err := proxy.Stop()
		require.NoError(t, err)
	}()

	proxy.InjectHeader("X-Test-Header", "test")
	proxy.InjectHeader("X-Test-Header2", "test2")

	// Test the proxy by making a request to it
	code, body := sendTestRequest(t, hostUrl, "/")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "{'test': 'value'}", string(body))

	// Send a request to the path that returns 404
	code, body = sendTestRequest(t, hostUrl, "/404")

	require.Equal(t, http.StatusNotFound, code)
	require.Contains(t, string(body), "Not Found")

	// Close the test server
	server.Close()

	code, body = sendTestRequest(t, hostUrl, "/")
	require.Equal(t, http.StatusInternalServerError, code)
	require.Contains(t, string(body), fmt.Sprintf("Error forwarding request: Get \"http://%s/\": dial tcp %s", serverAddr, serverAddr))
}

func TestProxyHandleWebSocket(t *testing.T) {
	server := testutil.NewWebsocketServer(t)
	defer server.Close()
	go func() {
		server.Start()
	}()

	proxy, addr := startProxy(t, server.Addr())
	defer func() {
		err := proxy.Stop()
		require.NoError(t, err)
	}()

	conn, resp, err := websocket.DefaultDialer.Dial("ws://"+addr, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	defer conn.Close()

	// Send a message to the server
	err = conn.WriteMessage(websocket.TextMessage, []byte("Hello from client"))
	require.NoError(t, err)

	// Receive message from the server
	_, message, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, "Message from client: Hello from client", string(message))

	// Send another message to the server
	err = conn.WriteMessage(websocket.TextMessage, []byte("Hello from client 2"))
	require.NoError(t, err)

	// Receive message from the server
	_, message, err = conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, "Message from client: Hello from client 2", string(message))

	// Close the server
	server.Close()

	// Send a message to the closed server
	err = conn.WriteMessage(websocket.TextMessage, []byte("Hello from client"))
	require.NoError(t, err)

	_, _, err = conn.ReadMessage()
	require.Error(t, err)
	potentialErrMessages := []string{
		"websocket: close 1006 (abnormal closure)",
		"An established connection was aborted by the software in your host machine",
		"connection reset by peer",
		"An existing connection was forcibly closed by the remote host",
	}
	found := false
	for _, msg := range potentialErrMessages {
		if strings.Contains(err.Error(), msg) {
			found = true
			break
		}
	}

	// If none of the expected error messages are found, fail the test
	if !found {
		t.Errorf("Expected one of the expected errors, got %s", err)
	}
}
