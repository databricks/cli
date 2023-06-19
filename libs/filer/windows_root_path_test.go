package filer

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWindowsRootPathForRoot(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("this test is meant for windows")
	}

	rp := NewWindowsRootPath("/")

	// Assert root value returned
	assert.Equal(t, "/", rp.Root())

	// case: absolute windows path
	path, err := rp.Join(`c:\a\b`)
	assert.NoError(t, err)
	assert.Equal(t, `c:\a\b`, path)

	// case: absolute windows path following file URI scheme
	path, err = rp.Join(`D:/a/b`)
	assert.NoError(t, err)
	assert.Equal(t, `D:/a/b`, path)

	// case: relative windows paths
	path, err = rp.Join(`c:a\b`)
	assert.NoError(t, err)
	assert.Equal(t, `c:a\b`, path)

	path, err = rp.Join(`c:a`)
	assert.NoError(t, err)
	assert.Equal(t, `c:a`, path)

	// case: relative windows paths following file URI scheme
	path, err = rp.Join(`c:a/b`)
	assert.NoError(t, err)
	assert.Equal(t, `C:a/b`, path)
}

func TestWindowsRootPath(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("this test is meant for windows")
	}

	tmpDir := t.TempDir()
	rp := NewWindowsRootPath(t.TempDir())

	// Assert root value returned
	assert.Equal(t, tmpDir, rp.Root())

	path, err := rp.Join(`a\b\c`)
	assert.NoError(t, err)
	assert.Equal(t, tmpDir+`\a\b`, path)

	path, err = rp.Join("a/b")
	assert.NoError(t, err)
	assert.Equal(t, tmpDir + `\a/b`, path)
}
