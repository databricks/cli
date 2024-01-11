package exec

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	shell, err := newShShell()
	assert.NoError(t, err)
	assert.NotNil(t, shell)
}

func TestShNotFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	t.Setenv("PATH", "")

	shell, err := newShShell()
	assert.NoError(t, err)
	assert.Nil(t, shell)
}
