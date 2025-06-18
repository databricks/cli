package exec

import (
	"os/exec"
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
	assert.Equal(t, "-c", opts.Args[1])
	assert.Equal(t, "echo hello", opts.Args[2])
}
