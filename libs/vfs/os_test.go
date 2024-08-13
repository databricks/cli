package vfs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOsNewWithRelativePath(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	p, err := New(".")
	require.NoError(t, err)
	require.Equal(t, wd, p.Native())
}

func TestOsPathParent(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	p := MustNew(wd)
	require.NotNil(t, p)

	// Traverse all the way to the root.
	for {
		q := p.Parent()
		if q == nil {
			// Parent returns nil when it is the root.
			break
		}

		p = q
	}

	// We should have reached the root.
	if runtime.GOOS == "windows" {
		require.Equal(t, filepath.VolumeName(wd)+`\`, p.Native())
	} else {
		require.Equal(t, "/", p.Native())
	}
}

func TestOsPathNative(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	p := MustNew(wd)
	require.NotNil(t, p)
	require.Equal(t, wd, p.Native())
}
