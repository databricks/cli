package apps

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsViteReady(t *testing.T) {
	t.Run("vite not running", func(t *testing.T) {
		// Assuming nothing is running on port 5173
		ready := isViteReady()
		assert.False(t, ready)
	})

	t.Run("vite is running", func(t *testing.T) {
		// Start a mock server on the Vite port
		listener, err := net.Listen("tcp", "localhost:"+vitePort)
		require.NoError(t, err)
		defer listener.Close()

		// Accept connections in the background
		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				conn.Close()
			}
		}()

		// Give the listener a moment to start
		time.Sleep(50 * time.Millisecond)

		ready := isViteReady()
		assert.True(t, ready)
	})
}

func TestCreateViteServerScript(t *testing.T) {
	scriptPath, err := createViteServerScript()
	require.NoError(t, err)
	defer os.Remove(scriptPath)

	// Verify the file exists
	_, err = os.Stat(scriptPath)
	assert.NoError(t, err)

	// Verify the file contains the embedded script
	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	assert.NotEmpty(t, content)

	// Verify it's a JavaScript file
	assert.Contains(t, string(content), "startViteServer")
}

func TestStartViteDevServerNoNode(t *testing.T) {
	// Skip this test if node is not available or in CI environments
	if os.Getenv("CI") != "" {
		t.Skip("Skipping node-dependent test in CI")
	}

	ctx := context.Background()
	ctx = cmdio.MockDiscard(ctx)

	// Create a temporary directory to act as project root
	tmpDir := t.TempDir()
	oldWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(oldWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create a client directory
	err = os.Mkdir("client", 0o755)
	require.NoError(t, err)

	// Try to start Vite server with invalid app URL (will fail fast)
	// This test mainly verifies the function signature and error handling
	_, _, _, err = startViteDevServer(ctx, "")
	assert.Error(t, err)
}

func TestVitePortConstant(t *testing.T) {
	assert.Equal(t, "5173", vitePort)
}

func TestViteServerScriptEmbedded(t *testing.T) {
	assert.NotEmpty(t, viteServerScript)

	scriptContent := string(viteServerScript)
	assert.Contains(t, scriptContent, "startViteServer")
	assert.Contains(t, scriptContent, "createServer")
	assert.Contains(t, scriptContent, "queriesHMRPlugin")
}
