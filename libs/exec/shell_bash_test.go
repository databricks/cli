package exec

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBashFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	shell, err := newBashShell()
	assert.NoError(t, err)
	assert.NotNil(t, shell)
}

func TestBashNotFound(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.SkipNow()
	}

	t.Setenv("PATH", "")

	shell, err := newBashShell()
	assert.NoError(t, err)
	assert.Nil(t, shell)
}
