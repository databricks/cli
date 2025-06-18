package exec

import (
	"os"
	"os/exec"
	"runtime"
	"testing"

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

func TestShellExecv_WindowsCleanup(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping windows test")
	}

	dir := t.TempDir()
	t.Setenv("TMPDIR", dir)

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
