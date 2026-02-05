package vite

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

func TestValidateDirPath(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	queriesDir := filepath.Join(tmpDir, "config", "queries")
	err = os.MkdirAll(queriesDir, 0o755)
	require.NoError(t, err)

	hiddenDir := filepath.Join(queriesDir, ".hidden")
	err = os.Mkdir(hiddenDir, 0o755)
	require.NoError(t, err)

	outsideDir := filepath.Join(tmpDir, "outside")
	err = os.Mkdir(outsideDir, 0o755)
	require.NoError(t, err)

	testFile := filepath.Join(queriesDir, "test.sql")
	err = os.WriteFile(testFile, []byte("SELECT 1"), 0o644)
	require.NoError(t, err)

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid directory path",
			path:        "config/queries",
			expectError: false,
		},
		{
			name:        "path outside allowed directory",
			path:        "outside",
			expectError: true,
			errorMsg:    "outside allowed directory",
		},
		{
			name:        "hidden directory",
			path:        "config/queries/.hidden",
			expectError: true,
			errorMsg:    "hidden directories are not allowed",
		},
		{
			name:        "path traversal attempt",
			path:        "config/queries/../../outside",
			expectError: true,
			errorMsg:    "outside allowed directory",
		},
		{
			name:        "file instead of directory",
			path:        "config/queries/test.sql",
			expectError: true,
			errorMsg:    "not a directory",
		},
		{
			name:        "prefix attack - similar directory name",
			path:        "config/queries-malicious",
			expectError: true,
			errorMsg:    "outside allowed directory",
		},
		{
			name:        "nonexistent directory",
			path:        "config/queries/nonexistent",
			expectError: true,
			errorMsg:    "failed to stat path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDirPath(tt.path)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBridgeHandleDirListRequest(t *testing.T) {
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	queriesDir := filepath.Join(tmpDir, "config", "queries")
	err = os.MkdirAll(queriesDir, 0o755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(queriesDir, "query1.sql"), []byte("SELECT 1"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(queriesDir, "query2.obo.sql"), []byte("SELECT 2"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(queriesDir, "readme.txt"), []byte("Not a SQL file"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(queriesDir, "config.json"), []byte("{}"), 0o644)
	require.NoError(t, err)

	subDir := filepath.Join(queriesDir, "subdir")
	err = os.Mkdir(subDir, 0o755)
	require.NoError(t, err)

	t.Run("only returns sql files", func(t *testing.T) {
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

		vb := NewBridge(ctx, w, "test-app", 5173)
		vb.tunnelConn = conn

		go func() { _ = vb.tunnelWriter(ctx) }()

		msg := &BridgeMessage{
			Type:      "dir:list",
			Path:      "config/queries",
			RequestID: "req-123",
		}

		err = vb.handleDirListRequest(msg)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		var response BridgeMessage
		err = json.Unmarshal(lastMessage, &response)
		require.NoError(t, err)

		assert.Equal(t, "dir:list:response", response.Type)
		assert.Equal(t, "req-123", response.RequestID)
		assert.Empty(t, response.Error)

		var files []string
		err = json.Unmarshal([]byte(response.Content), &files)
		require.NoError(t, err)

		assert.Len(t, files, 2)
		assert.Contains(t, files, "query1.sql")
		assert.Contains(t, files, "query2.obo.sql")
		assert.NotContains(t, files, "readme.txt")
		assert.NotContains(t, files, "config.json")
		assert.NotContains(t, files, "subdir")
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
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

		vb := NewBridge(ctx, w, "test-app", 5173)
		vb.tunnelConn = conn

		go func() { _ = vb.tunnelWriter(ctx) }()

		msg := &BridgeMessage{
			Type:      "dir:list",
			Path:      "../../etc",
			RequestID: "req-456",
		}

		err = vb.handleDirListRequest(msg)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		var response BridgeMessage
		err = json.Unmarshal(lastMessage, &response)
		require.NoError(t, err)

		assert.Equal(t, "dir:list:response", response.Type)
		assert.Equal(t, "req-456", response.RequestID)
		assert.NotEmpty(t, response.Error)
		assert.Contains(t, response.Error, "Invalid directory path")
	})
}
