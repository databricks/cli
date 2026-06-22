package internal

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPythonVersionOK(t *testing.T) {
	assert.True(t, pythonVersionOK("Python 3.11.0", "3.11"))
	assert.True(t, pythonVersionOK("Python 3.13.2", "3.11"))
	assert.False(t, pythonVersionOK("Python 3.10.6", "3.11"))
	assert.False(t, pythonVersionOK("garbage", "3.11"))
}

func TestEnsurePython(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not installed")
	}

	EnsurePython(t, "3.11")

	// After setup, the python3 resolved from PATH must satisfy the floor.
	out, err := exec.Command("python3", "-c", "import sys; print(sys.version_info >= (3, 11))").Output()
	require.NoError(t, err)
	assert.Equal(t, "True", strings.TrimSpace(string(out)))
}
