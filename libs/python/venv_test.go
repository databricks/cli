package python

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectVirtualEnvPath_NoVirtualEnvDetected(t *testing.T) {
	_, err := DetectVirtualEnvPath("testdata")
	assert.Equal(t, ErrNoVirtualEnvDetected, err)
}

func TestDetectVirtualEnvPath_invalid(t *testing.T) {
	_, err := DetectVirtualEnvPath("testdata/__invalid__")
	assert.Error(t, err)
}

func TestDetectVirtualEnvPath_wrongDir(t *testing.T) {
	_, err := DetectVirtualEnvPath("testdata/other-binaries-filtered")
	assert.Error(t, err)
}

func TestDetectVirtualEnvPath_happy(t *testing.T) {
	venv, err := DetectVirtualEnvPath("testdata/some-dir-with-venv")
	assert.NoError(t, err)
	found := "testdata/some-dir-with-venv/.venv"
	if runtime.GOOS == "windows" {
		found = "testdata\\some-dir-with-venv\\.venv"
	}
	assert.Equal(t, found, venv)
}
