package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchive(t *testing.T) {
	if os.Getenv("CLOUD_ENV") == "" {
		t.Skip("Skipping test in local environment")
	}

	t.Parallel()

	archiveDir := t.TempDir()
	binDir := t.TempDir()
	repoRoot := "../.."

	err := createArchive(archiveDir, binDir, repoRoot)
	require.NoError(t, err)

	assertDir := t.TempDir()
	err = extractTarGz(filepath.Join(archiveDir, "archive.tar.gz"), assertDir)
	require.NoError(t, err)

	// Go installation is a directory because it includes the
	// standard library source code along with the Go binary.
	assert.FileExists(t, filepath.Join(assertDir, "bin", "amd64", "go", "bin", "go"))
	assert.FileExists(t, filepath.Join(assertDir, "bin", "amd64", "uv"))
	assert.FileExists(t, filepath.Join(assertDir, "bin", "amd64", "jq"))

	// TODO: Uncomment these when we have arm64 support
	// in serverless clusters. Before that point we do not need
	// to download the arm64 binaries.
	// assert.FileExists(t, filepath.Join(assertDir, "bin", "arm64", "go", "bin", "go"))
	// assert.FileExists(t, filepath.Join(assertDir, "bin", "arm64", "uv"))
	// assert.FileExists(t, filepath.Join(assertDir, "bin", "arm64", "jq"))
	assert.NoFileExists(t, filepath.Join(assertDir, "bin", "arm64", "go"))
	assert.NoFileExists(t, filepath.Join(assertDir, "bin", "arm64", "uv"))
	assert.NoFileExists(t, filepath.Join(assertDir, "bin", "arm64", "jq"))

	assert.FileExists(t, filepath.Join(assertDir, "cli", "go.mod"))
	assert.FileExists(t, filepath.Join(assertDir, "cli", "go.sum"))
}
