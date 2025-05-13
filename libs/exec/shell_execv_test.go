package exec

import (
	"os/exec"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShellExecvOpts(t *testing.T) {
	opts := ExecvOptions{
		Env: []string{"key1=value1", "key2=value2"},
		Dir: "/a/b/c",
	}

	newOpts, err := shellExecvOpts("echo hello", opts)
	require.NoError(t, err)

	assert.Equal(t, []string{"key1=value1", "key2=value2"}, newOpts.Env)
	assert.Equal(t, "/a/b/c", newOpts.Dir)

	bashPath, err := exec.LookPath("bash")
	require.NoError(t, err)
	assert.Equal(t, bashPath, newOpts.Args[0])
	assert.Equal(t, "-e", newOpts.Args[1])
	assert.Equal(t, "echo hello", testutil.ReadFile(t, newOpts.Args[2]))
}
