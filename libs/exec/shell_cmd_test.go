package exec

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCmdFound(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	shell, err := newCmdShell()
	assert.NoError(t, err)
	assert.NotNil(t, shell)
}

func TestCmdNotFound(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.SkipNow()
	}

	t.Setenv("PATH", "")

	shell, err := newCmdShell()
	assert.NoError(t, err)
	assert.Nil(t, shell)
}
