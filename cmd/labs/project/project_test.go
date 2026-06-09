package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertEqualPaths(t *testing.T, expected, actual string) {
	expected = strings.ReplaceAll(expected, "/", string(os.PathSeparator))
	assert.Equal(t, expected, actual)
}

func TestLoad(t *testing.T) {
	ctx := t.Context()
	prj, err := Load(ctx, "testdata/installed-in-home/.databricks/labs/blueprint/lib/labs.yml")
	assert.NoError(t, err)
	assertEqualPaths(t, "testdata/installed-in-home/.databricks/labs/blueprint/lib", prj.folder)
}

func TestCheckUpdatesWithEmptyReleaseList(t *testing.T) {
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	rootDir, err := PathInLabs(ctx, "blueprint")
	require.NoError(t, err)
	prj := &Project{Name: "blueprint", rootDir: rootDir}
	require.NoError(t, prj.EnsureFoldersExist())

	// A repository without releases caches an empty list; the far-future
	// refresh timestamp keeps the cache valid so no network call is made.
	cache := []byte(`{"refreshed_at": "2033-01-01T00:00:00Z", "data": []}`)
	require.NoError(t, os.WriteFile(filepath.Join(prj.CacheDir(), "databrickslabs-blueprint-releases.json"), cache, ownerRW))
	version := []byte(`{"version": "v0.3.15", "date": "2023-10-24T15:04:05+01:00"}`)
	require.NoError(t, os.WriteFile(filepath.Join(prj.StateDir(), "version.json"), version, ownerRW))

	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	assert.NoError(t, prj.checkUpdates(cmd))
}
