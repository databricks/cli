package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO: Turn these tests off by default and add comment.
// Only run tests tests on a DBR environment CI job.
func TestUvDownloader(t *testing.T) {
	tmpDir := t.TempDir()

	for _, arch := range []string{"arm64", "amd64"} {
		err := uvDownloader{arch: arch, binDir: tmpDir}.Download()
		require.NoError(t, err)

		files, err := os.ReadDir(filepath.Join(tmpDir, arch))
		require.NoError(t, err)

		assert.Equal(t, 1, len(files))
		assert.Equal(t, "uv", files[0].Name())
	}
}

func TestJqDownloader(t *testing.T) {
	tmpDir := t.TempDir()

	for _, arch := range []string{"arm64", "amd64"} {
		err := jqDownloader{arch: arch, binDir: tmpDir}.Download()
		require.NoError(t, err)

		files, err := os.ReadDir(filepath.Join(tmpDir, arch))
		require.NoError(t, err)

		assert.Equal(t, 1, len(files))
		assert.Equal(t, "jq", files[0].Name())
	}
}
