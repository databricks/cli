package app

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	// Change to temp directory
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create the allowed directory
	queriesDir := filepath.Join(tmpDir, "config", "queries")
	err = os.MkdirAll(queriesDir, 0o755)
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
			err := validateFilePath(tt.path)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestViteBridgeMessageSerialization(t *testing.T) {
	tests := []struct {
		name string
		msg  ViteBridgeMessage
	}{
		{
			name: "tunnel ready message",
			msg: ViteBridgeMessage{
				Type:     "tunnel:ready",
				TunnelID: "test-tunnel-123",
			},
		},
		{
			name: "fetch request message",
			msg: ViteBridgeMessage{
				Type:      "fetch",
				Path:      "/src/components/ui/card.tsx",
				Method:    "GET",
				RequestID: "req-123",
			},
		},
		{
			name: "connection request message",
			msg: ViteBridgeMessage{
				Type:      "connection:request",
				Viewer:    "user@example.com",
				RequestID: "req-456",
			},
		},
		{
			name: "fetch response with headers",
			msg: ViteBridgeMessage{
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

			var decoded ViteBridgeMessage
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

func TestViteBridgeHandleMessage(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())

	w := &databricks.WorkspaceClient{}

	vb := NewViteBridge(ctx, w, "test-app", 5173)

	tests := []struct {
		name        string
		msg         *ViteBridgeMessage
		expectError bool
	}{
		{
			name: "tunnel ready message",
			msg: &ViteBridgeMessage{
				Type:     "tunnel:ready",
				TunnelID: "tunnel-123",
			},
			expectError: false,
		},
		{
			name: "unknown message type",
			msg: &ViteBridgeMessage{
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

func TestViteBridgeHandleFileReadRequest(t *testing.T) {
	// Create a temporary directory structure
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	queriesDir := filepath.Join(tmpDir, "config", "queries")
	err = os.MkdirAll(queriesDir, 0o755)
	require.NoError(t, err)

	testContent := "SELECT * FROM users WHERE id = 1"
	testFile := filepath.Join(queriesDir, "test_query.sql")
	err = os.WriteFile(testFile, []byte(testContent), 0o644)
	require.NoError(t, err)

	t.Run("successful file read", func(t *testing.T) {
		ctx := cmdio.MockDiscard(context.Background())
		w := &databricks.WorkspaceClient{}

		// Create a mock tunnel connection using httptest
		var lastMessage []byte
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
			lastMessage = message
		}))
		defer server.Close()

		// Connect to the mock server
		wsURL := "ws" + server.URL[4:]
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		defer conn.Close()

		vb := NewViteBridge(ctx, w, "test-app", 5173)
		vb.tunnelConn = conn

		go func() { _ = vb.tunnelWriter(ctx) }()

		msg := &ViteBridgeMessage{
			Type:      "file:read",
			Path:      "config/queries/test_query.sql",
			RequestID: "req-123",
		}

		err = vb.handleFileReadRequest(msg)
		require.NoError(t, err)

		// Give the message time to be sent
		time.Sleep(100 * time.Millisecond)

		// Parse the response
		var response ViteBridgeMessage
		err = json.Unmarshal(lastMessage, &response)
		require.NoError(t, err)

		assert.Equal(t, "file:read:response", response.Type)
		assert.Equal(t, "req-123", response.RequestID)
		assert.Equal(t, testContent, response.Content)
		assert.Empty(t, response.Error)
	})

	t.Run("file not found", func(t *testing.T) {
		ctx := cmdio.MockDiscard(context.Background())
		w := &databricks.WorkspaceClient{}

		var lastMessage []byte
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
			lastMessage = message
		}))
		defer server.Close()

		wsURL := "ws" + server.URL[4:]
		conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
		require.NoError(t, err)
		defer resp.Body.Close()
		defer conn.Close()

		vb := NewViteBridge(ctx, w, "test-app", 5173)
		vb.tunnelConn = conn

		go func() { _ = vb.tunnelWriter(ctx) }()

		msg := &ViteBridgeMessage{
			Type:      "file:read",
			Path:      "config/queries/nonexistent.sql",
			RequestID: "req-456",
		}

		err = vb.handleFileReadRequest(msg)
		require.NoError(t, err)

		// Give the message time to be sent
		time.Sleep(100 * time.Millisecond)

		var response ViteBridgeMessage
		err = json.Unmarshal(lastMessage, &response)
		require.NoError(t, err)

		assert.Equal(t, "file:read:response", response.Type)
		assert.Equal(t, "req-456", response.RequestID)
		assert.NotEmpty(t, response.Error)
	})
}

func TestViteBridgeStop(t *testing.T) {
	ctx := cmdio.MockDiscard(context.Background())
	w := &databricks.WorkspaceClient{}

	vb := NewViteBridge(ctx, w, "test-app", 5173)

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

func TestNewViteBridge(t *testing.T) {
	ctx := context.Background()
	w := &databricks.WorkspaceClient{}
	appName := "test-app"

	vb := NewViteBridge(ctx, w, appName, 5173)

	assert.NotNil(t, vb)
	assert.Equal(t, appName, vb.appName)
	assert.NotNil(t, vb.httpClient)
	assert.NotNil(t, vb.stopChan)
	assert.NotNil(t, vb.connectionRequests)
	assert.Equal(t, 10, cap(vb.connectionRequests))
}
