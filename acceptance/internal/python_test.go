package internal

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureModernPython(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not installed")
	}

	EnsureModernPython(t)

	// After setup, the python3 resolved from PATH must satisfy the floor.
	out, err := exec.Command("python3", "-c", "import sys; print(sys.version_info >= (3, 11))").Output()
	require.NoError(t, err)
	assert.Equal(t, "True", strings.TrimSpace(string(out)))
}
