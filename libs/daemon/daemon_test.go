package daemon

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDaemon(t *testing.T) {
	tmpDir := t.TempDir()
	cmd := exec.Command("go", "run", "internal/parent_process/main.go", tmpDir)

	// cmd.Run() will block until the parent process exits.
	err := cmd.Run()
	require.NoError(t, err)

	// Assert that a PID file was created for the child process.
	assert.FileExists(t, filepath.Join(tmpDir, "child.pid"))

	// Wait 10 seconds for the server to start and to write the port number to
	// a file.
	portFilePath := filepath.Join(tmpDir, "port.txt")
	assert.Eventually(t, func() bool {
		_, err := os.Stat(portFilePath)
		return err == nil
	}, 10*time.Second, 100*time.Millisecond)

	port, err := strconv.Atoi(testutil.ReadFile(t, portFilePath))
	require.NoError(t, err)

	// Query the local server, which should be alive even after the parent process
	// has exited.
	r, err := http.Get("http://localhost:" + strconv.Itoa(port))
	require.NoError(t, err)
	defer r.Body.Close()

	// The server should respond with "child says hi".
	assert.Equal(t, http.StatusOK, r.StatusCode)
	b, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	assert.Equal(t, "child says hi", string(b))
}
