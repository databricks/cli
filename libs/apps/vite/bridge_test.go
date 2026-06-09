package vite

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFilePath(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Create the allowed directory
	queriesDir := filepath.Join(tmpDir, "config", "queries")
	err := os.MkdirAll(queriesDir, 0o755)
	require.NoError(t, err)

	// Create a valid test file
	validFile := filepath.Join(queriesDir, "test.sql")
	err = os.WriteFile(validFile, []byte("SELECT * FROM table"), 0o644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid file path",
			path:        "config/queries/test.sql",
			expectError: false,
		},
		{
			name:        "path outside allowed directory",
			path:        "../../etc/passwd",
			expectError: true,
			errorMsg:    "outside allowed directory",
		},
		{
			name:        "wrong file extension",
			path:        "config/queries/test.txt",
			expectError: true,
			errorMsg:    "only .sql files are allowed",
		},
		{
			name:        "hidden file",
			path:        "config/queries/.hidden.sql",
			expectError: true,
			errorMsg:    "hidden files are not allowed",
		},
		{
			name:        "path traversal attempt",
			path:        "config/queries/../../../etc/passwd",
			expectError: true,
			errorMsg:    "outside allowed directory",
		},
		{
			name:        "prefix attack - similar directory name",
			path:        "config/queries-malicious/test.sql",
			expectError: true,
			errorMsg:    "outside allowed directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFilePath(tt.path)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBridgeMessageSerialization(t *testing.T) {
	tests := []struct {
		name string
		msg  BridgeMessage
	}{
		{
			name: "tunnel ready message",
			msg: BridgeMessage{
				Type:     "tunnel:ready",
				TunnelID: "test-tunnel-123",
			},
		},
		{
			name: "fetch request message",
			msg: BridgeMessage{
				Type:      "fetch",
				Path:      "/src/components/ui/card.tsx",
				Method:    "GET",
				RequestID: "req-123",
			},
		},
		{
			name: "connection request message",
			msg: BridgeMessage{
				Type:      "connection:request",
				Viewer:    "user@example.com",
				RequestID: "req-456",
			},
		},
		{
			name: "fetch response with headers",
			msg: BridgeMessage{
				Type:   "fetch:response:meta",
				Status: 200,
				Headers: map[string]any{
					"Content-Type": "application/json",
				},
				RequestID: "req-789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.msg)
			require.NoError(t, err)

			var decoded BridgeMessage
			err = json.Unmarshal(data, &decoded)
			require.NoError(t, err)

			assert.Equal(t, tt.msg.Type, decoded.Type)
			assert.Equal(t, tt.msg.TunnelID, decoded.TunnelID)
			assert.Equal(t, tt.msg.Path, decoded.Path)
			assert.Equal(t, tt.msg.Method, decoded.Method)
			assert.Equal(t, tt.msg.RequestID, decoded.RequestID)
		})
	}
}

func TestBridgeHandleMessage(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())

	w := &databricks.WorkspaceClient{}

	vb := NewBridge(ctx, w, "test-app", 5173, false)

	tests := []struct {
		name        string
		msg         *BridgeMessage
		expectError bool
	}{
		{
			name: "tunnel ready message",
			msg: &BridgeMessage{
				Type:     "tunnel:ready",
				TunnelID: "tunnel-123",
			},
			expectError: false,
		},
		{
			name: "unknown message type",
			msg: &BridgeMessage{
				Type: "unknown:type",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := vb.handleMessage(tt.msg)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.msg.Type == "tunnel:ready" {
				assert.Equal(t, tt.msg.TunnelID, vb.tunnelID)
			}
		})
	}
}

// waitForMessage waits for the websocket test server to deliver a message.
// Tests must synchronize on a channel (not a sleep) so they pass under -race.
func waitForMessage(t *testing.T, received <-chan []byte) []byte {
	select {
	case message := <-received:
		return message
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for message")
		return nil
	}
}

func TestBridgeHandleFileReadRequest(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	queriesDir := filepath.Join(tmpDir, "config", "queries")
	err := os.MkdirAll(queriesDir, 0o755)
	require.NoError(t, err)

	testContent := "SELECT * FROM users WHERE id = 1"
	testFile := filepath.Join(queriesDir, "test_query.sql")
	err = os.WriteFile(testFile, []byte(testContent), 0o644)
	require.NoError(t, err)

	t.Run("successful file read", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		w := &databricks.WorkspaceClient{}

		// Create a mock tunnel connection using httptest
		received := make(chan []byte, 1)
		upgrader := websocket.Upgrader{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Errorf("failed to upgrade: %v", err)
				return
			}
			defer conn.Close()

			// Read the message sent by handleFileReadRequest
			_, message, err := conn.ReadMessage()
			if err != nil {
				t.Errorf("failed to read message: %v", err)
				return
			}
			received <- message
		}))
		defer server.Close()

		// Connect to the mock server
		wsURL := "ws" + server.URL[4:]
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		defer conn.Close()

		vb := NewBridge(ctx, w, "test-app", 5173, false)
		vb.tunnelConn.Store(conn)

		go func() { _ = vb.tunnelWriter(ctx) }()

		msg := &BridgeMessage{
			Type:      "file:read",
			Path:      "config/queries/test_query.sql",
			RequestID: "req-123",
		}

		err = vb.handleFileReadRequest(msg)
		require.NoError(t, err)

		// Parse the response
		var response BridgeMessage
		err = json.Unmarshal(waitForMessage(t, received), &response)
		require.NoError(t, err)

		assert.Equal(t, "file:read:response", response.Type)
		assert.Equal(t, "req-123", response.RequestID)
		assert.Equal(t, testContent, response.Content)
		assert.Empty(t, response.Error)
	})

	t.Run("file not found", func(t *testing.T) {
		ctx := cmdio.MockDiscard(t.Context())
		w := &databricks.WorkspaceClient{}

		received := make(chan []byte, 1)
		upgrader := websocket.Upgrader{}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				t.Errorf("failed to upgrade: %v", err)
				return
			}
			defer conn.Close()

			_, message, err := conn.ReadMessage()
			if err != nil {
				t.Errorf("failed to read message: %v", err)
				return
			}
			received <- message
		}))
		defer server.Close()

		wsURL := "ws" + server.URL[4:]
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		defer conn.Close()

		vb := NewBridge(ctx, w, "test-app", 5173, false)
		vb.tunnelConn.Store(conn)

		go func() { _ = vb.tunnelWriter(ctx) }()

		msg := &BridgeMessage{
			Type:      "file:read",
			Path:      "config/queries/nonexistent.sql",
			RequestID: "req-456",
		}

		err = vb.handleFileReadRequest(msg)
		require.NoError(t, err)

		var response BridgeMessage
		err = json.Unmarshal(waitForMessage(t, received), &response)
		require.NoError(t, err)

		assert.Equal(t, "file:read:response", response.Type)
		assert.Equal(t, "req-456", response.RequestID)
		assert.NotEmpty(t, response.Error)
	})
}

func TestBridgeStop(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := &databricks.WorkspaceClient{}

	vb := NewBridge(ctx, w, "test-app", 5173, false)

	// Call Stop multiple times to ensure it's idempotent
	vb.Stop()
	vb.Stop()
	vb.Stop()

	// Verify stopChan is closed
	select {
	case <-vb.stopChan:
		// Channel is closed, this is expected
	default:
		t.Error("stopChan should be closed after Stop()")
	}
}

func TestNewBridge(t *testing.T) {
	ctx := t.Context()
	w := &databricks.WorkspaceClient{}
	appName := "test-app"

	vb := NewBridge(ctx, w, appName, 5173, false)

	assert.NotNil(t, vb)
	assert.Equal(t, appName, vb.appName)
	assert.NotNil(t, vb.httpClient)
	assert.NotNil(t, vb.stopChan)
	assert.NotNil(t, vb.connectionRequests)
	assert.Equal(t, 10, cap(vb.connectionRequests))
	assert.False(t, vb.autoApprove)
}

func TestNewBridge_AutoApprove(t *testing.T) {
	ctx := t.Context()
	w := &databricks.WorkspaceClient{}

	vb := NewBridge(ctx, w, "test-app", 5173, true)

	assert.NotNil(t, vb)
	assert.True(t, vb.autoApprove)
}

// newWSConn starts a websocket server that discards inbound messages and
// returns a client connection to it.
func newWSConn(t *testing.T) *websocket.Conn {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
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
	t.Cleanup(server.Close)

	wsURL := "ws" + server.URL[4:]
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	resp.Body.Close()
	t.Cleanup(func() { conn.Close() })
	return conn
}

func TestBridgeSetTunnelConnSwapDuringWrites(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := &databricks.WorkspaceClient{}

	conn1 := newWSConn(t)
	conn2 := newWSConn(t)

	vb := NewBridge(ctx, w, "test-app", 5173, false)
	vb.tunnelConn.Store(conn1)

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		// The writer may return an error if a queued write hits the
		// just-closed old connection; this test only checks that the swap is
		// race-free and closes the old connection.
		_ = vb.tunnelWriter(ctx)
	}()

	for i := range 100 {
		vb.tunnelWriteChan <- prioritizedMessage{
			messageType: websocket.TextMessage,
			data:        []byte("payload"),
			priority:    1,
		}
		if i == 50 {
			// Simulate the reconnect path swapping in a fresh connection.
			vb.setTunnelConn(conn2)
		}
	}

	close(vb.stopChan)
	select {
	case <-writerDone:
	case <-time.After(5 * time.Second):
		t.Fatal("tunnel writer did not stop")
	}

	// setTunnelConn must close the connection it replaced.
	err := conn1.WriteMessage(websocket.TextMessage, []byte("x"))
	require.Error(t, err)
}

func TestBridgeConnectionRequestSendDoesNotBlockAfterStop(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := &databricks.WorkspaceClient{}

	vb := NewBridge(ctx, w, "test-app", 5173, false)

	// Fill the queue so an unguarded send would block forever.
	for range cap(vb.connectionRequests) {
		vb.connectionRequests <- &BridgeMessage{Type: "connection:request"}
	}

	vb.Stop()

	done := make(chan error, 1)
	go func() {
		done <- vb.handleMessage(&BridgeMessage{Type: "connection:request"})
	}()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("handleMessage blocked on a full connectionRequests queue after stop")
	}
}

func TestBridgeConnectionRequestSequentialPrompts(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := &databricks.WorkspaceClient{}

	vb := NewBridge(ctx, w, "test-app", 5173, false)
	go vb.readStdinLines(strings.NewReader("y\nn\n"))

	// Both prompts must get their own line; the old per-prompt reader leaked a
	// goroutine on timeout that could swallow the answer to the next prompt.
	require.NoError(t, vb.handleConnectionRequest(&BridgeMessage{Type: "connection:request", Viewer: "a@example.com", RequestID: "req-1"}))
	require.NoError(t, vb.handleConnectionRequest(&BridgeMessage{Type: "connection:request", Viewer: "b@example.com", RequestID: "req-2"}))

	var responses []BridgeMessage
	for range 2 {
		msg := <-vb.tunnelWriteChan
		var response BridgeMessage
		require.NoError(t, json.Unmarshal(msg.data, &response))
		responses = append(responses, response)
	}

	assert.Equal(t, "connection:response", responses[0].Type)
	assert.Equal(t, "req-1", responses[0].RequestID)
	assert.True(t, responses[0].Approved)
	assert.Equal(t, "connection:response", responses[1].Type)
	assert.Equal(t, "req-2", responses[1].RequestID)
	assert.False(t, responses[1].Approved)
}

func TestBridgeHandleConnectionRequest_AutoApproveSkipsStdin(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	w := &databricks.WorkspaceClient{}

	received := make(chan []byte, 1)
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("failed to upgrade: %v", err)
			return
		}
		defer conn.Close()

		_, message, err := conn.ReadMessage()
		if err != nil {
			t.Errorf("failed to read message: %v", err)
			return
		}
		received <- message
	}))
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	require.NoError(t, err)
	defer resp.Body.Close()
	defer conn.Close()

	vb := NewBridge(ctx, w, "test-app", 5173, true)
	vb.tunnelConn.Store(conn)

	go func() { _ = vb.tunnelWriter(ctx) }()

	msg := &BridgeMessage{
		Type:      "connection:request",
		Viewer:    "alice@example.com",
		RequestID: "req-auto",
	}

	require.NoError(t, vb.handleConnectionRequest(msg))

	var response BridgeMessage
	require.NoError(t, json.Unmarshal(waitForMessage(t, received), &response))
	assert.Equal(t, "connection:response", response.Type)
	assert.Equal(t, "req-auto", response.RequestID)
	assert.True(t, response.Approved)
}
