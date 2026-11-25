package agents

import (
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveBinaryNotOnPath(t *testing.T) {
	stubLookPath(t)
	_, err := ResolveBinary(&Agent{Binary: "claude", DisplayName: "Claude Code"})
	assert.ErrorIs(t, err, exec.ErrNotFound)
}

func TestResolveBinaryRefusesDotRelativeBinary(t *testing.T) {
	orig := LookPath
	LookPath = func(name string) (string, error) { return filepath.Join(".", name), exec.ErrDot }
	t.Cleanup(func() { LookPath = orig })

	_, err := ResolveBinary(&Agent{Binary: "claude", DisplayName: "Claude Code"})
	assert.ErrorIs(t, err, exec.ErrDot)
}
