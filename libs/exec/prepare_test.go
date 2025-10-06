package exec

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrepareShellCommand(t *testing.T) {
	prep, err := PrepareShellCommand("echo hello")
	require.NoError(t, err)

	bashPath, err := exec.LookPath("bash")
	require.NoError(t, err)
	assert.Equal(t, bashPath, prep.Executable)
	assert.Equal(t, bashPath, prep.Args[0])
	assert.Equal(t, "-ec", prep.Args[1])
	assert.Equal(t, "echo hello", prep.Args[2])
	assert.Nil(t, prep.CleanupFn)
}
