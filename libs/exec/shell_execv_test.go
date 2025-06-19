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

func TestShellExecv_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("skipping windows test")
	}

	cmdExePath, err := exec.LookPath("cmd.exe")
	require.NoError(t, err)

	// Cleanup environment so that other shells like bash and sh are not used.
	testutil.NullEnvironment(t)

	// Configure PATH so that only cmd.exe shows up.
	binDir := t.TempDir()
	testutil.CopyFile(t, cmdExePath, filepath.Join(binDir, "cmd.exe"))
	os.Setenv("PATH", binDir)

	tests := []struct {
		name     string
		content  string
		exitCode int
	}{
		{name: "success", content: "echo hello", exitCode: 0},
		{name: "non-zero exit", content: "exit 127", exitCode: 127},
		{name: "command error", content: "not-a-command", exitCode: 1},
	}

	for _, test := range tests {
		dir := t.TempDir()
		t.Setenv("TMP", dir)

		opts, err := shellExecvOpts(test.content, dir, []string{})
		require.NoError(t, err)

		// Verify that the temporary file is created.
		files, err := os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.Regexp(t, "cli-exec.*\\.cmd", files[0].Name())

		exitCode := -1
		opts.windowsExit = func(status int) {
			exitCode = status
		}

		// Execute the script.
		err = Execv(opts)
		require.NoError(t, err)

		// Verify that the temporary file is cleaned up after execution.
		files, err = os.ReadDir(dir)
		require.NoError(t, err)
		assert.Len(t, files, 0)

		// Verify that CLI would exit with the correct exit code.
		assert.Equal(t, test.exitCode, exitCode)
	}
}
