package appproxy

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/require"

	"github.com/gorilla/websocket"
)

const (
	PROXY_PORT = 8081
)

var (
	PROXY_ADDR = fmt.Sprintf("localhost:%d", PROXY_PORT)
	PROXY_URL  = "http://" + PROXY_ADDR
)

func sendTestRequest(t *testing.T, path string) (int, []byte) {
	req, err := http.NewRequest("GET", PROXY_URL+path, bytes.NewBufferString("{'test': 'value'}"))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, body
}

func startProxy(t *testing.T, serverAddr string) *Proxy {
	proxy, err := New("http://" + serverAddr)
	require.NoError(t, err)

	ln, err := proxy.Listen(PROXY_ADDR)
	require.NoError(t, err)

	go func() {
		_ = proxy.Serve(ln)
	}()

	return proxy
}

func TestProxyStart(t *testing.T) {
	server := testserver.NewHttpServer(t, map[string]string{
		"Content-Type":   "application/json",
		"X-Test-Header":  "test",
		"X-Test-Header2": "test2",
	})

	go func() {
		server.Start()
	}()

	serverAddr := server.Listener.Addr().String()
	proxy := startProxy(t, serverAddr)
	defer func() {
		err := proxy.Stop()
		require.NoError(t, err)
	}()

	proxy.InjectHeader("X-Test-Header", "test")
	proxy.InjectHeader("X-Test-Header2", "test2")

	// Test the proxy by making a request to it
	code, body := sendTestRequest(t, "/")
	require.Equal(t, http.StatusOK, code)
	require.Equal(t, "{'test': 'value'}", string(body))

	// Send a request to the path that returns 404
	code, body = sendTestRequest(t, "/404")

	require.Equal(t, http.StatusNotFound, code)
	require.Contains(t, string(body), "Not Found")

	// Close the test server
	server.Close()

	code, body = sendTestRequest(t, "/")
	require.Equal(t, http.StatusInternalServerError, code)
	require.Contains(t, string(body), fmt.Sprintf("Error forwarding request: Get \"http://%s/\": dial tcp %s", serverAddr, serverAddr))
}

func TestProxyHandleWebSocket(t *testing.T) {
	server := testserver.NewWebsocketServer(t)
	defer server.Close()
	go func() {
		server.Start()
	}()

	proxy := startProxy(t, server.Addr())
	defer func() {
		err := proxy.Stop()
		require.NoError(t, err)
	}()

	conn, resp, err := websocket.DefaultDialer.Dial("ws://"+PROXY_ADDR, nil)
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
	if !strings.Contains(err.Error(), "websocket: close 1006 (abnormal closure)") &&
		!strings.Contains(err.Error(), "An established connection was aborted by the software in your host machine") {
		t.Errorf("Expected abnormal closure or An established connection was aborted, got %s", err)
	}
}
