package apps

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/databricks/cli/libs/apps/vite"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsViteReady(t *testing.T) {
	t.Run("vite not running", func(t *testing.T) {
		// Assuming nothing is running on port 5173
		ready := isViteReady(5173)
		assert.False(t, ready)
	})

	t.Run("vite is running", func(t *testing.T) {
		// Use a free port so the check cannot collide with anything else bound to 5173.
		listener, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		defer listener.Close()

		go func() {
			for {
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				conn.Close()
			}
		}()

		// A single dial has only viteReadyCheckInterval (100ms) to succeed. On loaded
		// CI runners, and on Windows where dialing "localhost" may first try an
		// unbound ::1 before 127.0.0.1, one attempt can time out. Poll like the
		// production wait loop does.
		port := listener.Addr().(*net.TCPAddr).Port
		assert.Eventually(t, func() bool { return isViteReady(port) }, 10*time.Second, 50*time.Millisecond)
	})
}

func TestViteServerScriptContent(t *testing.T) {
	// Verify the embedded script is not empty
	assert.NotEmpty(t, vite.ServerScript)

	// Verify it's a JavaScript file with expected content
	assert.Contains(t, string(vite.ServerScript), "startViteServer")
}

func TestStartViteDevServerNoNode(t *testing.T) {
	// Skip this test if node is not available or in CI environments
	if os.Getenv("CI") != "" {
		t.Skip("Skipping node-dependent test in CI")
	}

	ctx := t.Context()
	ctx = cmdio.MockDiscard(ctx)

	// Create a temporary directory to act as project root
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Create a client directory
	err := os.Mkdir("client", 0o755)
	require.NoError(t, err)

	// Try to start Vite server with invalid app URL (will fail fast)
	// This test mainly verifies the function signature and error handling
	_, _, err = startViteDevServer(ctx, "", 5173)
	assert.Error(t, err)
}

func TestViteServerScriptEmbedded(t *testing.T) {
	assert.NotEmpty(t, vite.ServerScript)

	scriptContent := string(vite.ServerScript)
	assert.Contains(t, scriptContent, "startViteServer")
	assert.Contains(t, scriptContent, "createServer")
	assert.Contains(t, scriptContent, "queriesHMRPlugin")
}
