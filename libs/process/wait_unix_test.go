package process

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWaitUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping test on Windows")
	}

	t.Parallel()
	tmpDir := t.TempDir()

	pidFile := filepath.Join(tmpDir, "child.pid")
	outputFile := filepath.Join(tmpDir, "output.txt")

	// For this test, we cannot start the background process in this test itself
	// and instead have to use parent.sh as an intermediary.
	//
	// This is because in Unix if we start the background process in this test itself,
	// the background process will be a child of this test process and thus would
	// need to be reaped by this test process (using the Wait function / syscall).
	// Otherwise waitForPid will forever wait for the background process to finish.
	//
	// If we rely on an intermediate script to start the background process, the
	// background process is reasigned to the init process (PID 1) once the parent
	// exits and thus we can successfully wait for it in this test using waitForPid function.
	cmd := exec.Command("./testdata/parent.sh", pidFile, outputFile)
	err := cmd.Start()
	require.NoError(t, err)

	// Wait 5 seconds for the parent bash script to write the child's PID to the file.
	var childPid int
	require.Eventually(t, func() bool {
		b, err := os.ReadFile(pidFile)
		if err != nil {
			return false
		}

		childPid, err = strconv.Atoi(string(b))
		require.NoError(t, err)
		return true
	}, 2*time.Second, 100*time.Millisecond)

	// The output file should not exist yet since the background process should
	// still be running.
	assert.NoFileExists(t, outputFile)

	// Wait for the background process to finish.
	err = waitForPid(childPid)
	assert.NoError(t, err)

	// The output file should exist now since the background process has finished.
	testutil.AssertFileContents(t, outputFile, "abc\n")

	// Since the background process has finished, waiting for it again should
	// return an error.
	err = waitForPid(childPid)
	assert.Regexp(t, "process with pid .* does not exist", err.Error())
}
