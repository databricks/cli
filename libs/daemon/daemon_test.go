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

// This test is inherently flaky because it relies on a timeout to wait for
// the server process to start. While it's expected to be rare, we have evidence of
// this triggering in CI (where this failure is more likely due to machine virtualization)
// ref: https://github.com/databricks/cli/pull/2526#issuecomment-2739524677
//
// We can trigger this test back on once we actually start using the daemon package.
// That may never happen in which case we can remove this entire package at some point
// in the future.
// Date this test was disabled: 20th March 2025.
func DisabledTestDaemon(t *testing.T) {
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
