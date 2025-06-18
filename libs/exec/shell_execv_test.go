package exec

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellExecvOpts(t *testing.T) {
	opts, err := shellExecvOpts("echo hello", "/a/b/c", []string{"key1=value1", "key2=value2"})
	require.NoError(t, err)

	assert.Equal(t, []string{"key1=value1", "key2=value2"}, opts.Env)
	assert.Equal(t, "/a/b/c", opts.Dir)

	bashPath, err := exec.LookPath("bash")
	require.NoError(t, err)
	assert.Equal(t, bashPath, opts.Args[0])
	assert.Equal(t, "-ec", opts.Args[1])
	assert.Equal(t, "echo hello", opts.Args[2])
}

// TODO: Add cases for other cases, like non 0 exit and the command itself erroring.
func TestShellExecv_WindowsCleanup(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping windows test")
	}

	cmdExePath, err := exec.LookPath("cmd.exe")
	require.NoError(t, err)

	// Cleanup environment so that other shells like bash and sh are not used.
	testutil.NullEnvironment(t)

	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)

	// Configure PATH so that only cmd.exe shows up.
	binDir := t.TempDir()
	testutil.CopyFile(t, cmdExePath, filepath.Join(binDir, "cmd.exe"))
	os.Setenv("PATH", binDir)

	content := "echo hello"
	opts, err := shellExecvOpts(content, dir, []string{})
	require.NoError(t, err)

	// Verify that the temporary file is created.
	files, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, files, 1)
	assert.Regexp(t, "cli-exec.*\\.cmd", files[0].Name())

	// Execute the script.
	err = Execv(opts)
	require.NoError(t, err)

	// Verify that the temporary file is cleaned up after execution.
	files, err = os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, files, 0)
}
