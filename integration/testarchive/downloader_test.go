package testarchive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testarchive"
	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUvDownloader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	t.Parallel()

	testutil.GetEnvOrSkipTest(t, "CLOUD_ENV")

	tmpDir := t.TempDir()

	for _, arch := range []string{"arm64", "amd64"} {
		err := testarchive.UvDownloader{Arch: arch, BinDir: tmpDir}.Download()
		require.NoError(t, err)

		files, err := os.ReadDir(filepath.Join(tmpDir, arch))
		require.NoError(t, err)

		assert.Equal(t, 1, len(files))
		assert.Equal(t, "uv", files[0].Name())
	}
}

func TestJqDownloader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	t.Parallel()

	testutil.GetEnvOrSkipTest(t, "CLOUD_ENV")

	tmpDir := t.TempDir()

	for _, arch := range []string{"arm64", "amd64"} {
		err := testarchive.JqDownloader{Arch: arch, BinDir: tmpDir}.Download()
		require.NoError(t, err)

		files, err := os.ReadDir(filepath.Join(tmpDir, arch))
		require.NoError(t, err)

		assert.Equal(t, 1, len(files))
		assert.Equal(t, "jq", files[0].Name())
	}
}

func TestGoDownloader(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}

	t.Parallel()

	testutil.GetEnvOrSkipTest(t, "CLOUD_ENV")

	tmpDir := t.TempDir()

	for _, arch := range []string{"arm64", "amd64"} {
		err := testarchive.GoDownloader{Arch: arch, BinDir: tmpDir, RepoRoot: "../.."}.Download()
		require.NoError(t, err)

		entries, err := os.ReadDir(filepath.Join(tmpDir, arch))
		require.NoError(t, err)

		assert.Equal(t, 1, len(entries))
		assert.Equal(t, "go", entries[0].Name())
		assert.True(t, entries[0].IsDir())

		binaryPath := filepath.Join(tmpDir, arch, "go", "bin", "go")
		_, err = os.Stat(binaryPath)
		require.NoError(t, err)
	}
}
