package fs

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotSpecifyingVolumeForWindowsPathErrors(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip()
	}

	ctx := context.Background()
	pathWithVolume := `file:/c:/foo/bar`
	pathWOVolume := `file:/uno/dos`

	_, path, err := filerForPath(ctx, pathWithVolume)
	assert.NoError(t, err)
	assert.Equal(t, `/foo/bar`, path)

	_, _, err = filerForPath(ctx, pathWOVolume)
	assert.Equal(t, "no volume specified for path: uno/dos", err.Error())
}
