package sync

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncOptionsFromBundle(t *testing.T) {
	tempDir := t.TempDir()
	b := &bundle.Bundle{
		Config: config.Root{
			Path: tempDir,

			Bundle: config.Bundle{
				Target: "default",
			},

			Workspace: &config.Workspace{
				FilesPath: "/Users/jane@doe.com/path",
			},
		},
	}

	f := syncFlags{}
	opts, err := f.syncOptionsFromBundle(New(), []string{}, b)
	require.NoError(t, err)
	assert.Equal(t, tempDir, opts.LocalPath)
	assert.Equal(t, "/Users/jane@doe.com/path", opts.RemotePath)
	assert.Equal(t, filepath.Join(tempDir, ".databricks", "bundle", "default"), opts.SnapshotBasePath)
	assert.NotNil(t, opts.WorkspaceClient)
}

func TestSyncOptionsFromArgsRequiredTwoArgs(t *testing.T) {
	var err error
	f := syncFlags{}
	_, err = f.syncOptionsFromArgs(New(), []string{})
	require.ErrorIs(t, err, flag.ErrHelp)
	_, err = f.syncOptionsFromArgs(New(), []string{"foo"})
	require.ErrorIs(t, err, flag.ErrHelp)
	_, err = f.syncOptionsFromArgs(New(), []string{"foo", "bar", "qux"})
	require.ErrorIs(t, err, flag.ErrHelp)
}

func TestSyncOptionsFromArgs(t *testing.T) {
	f := syncFlags{}
	opts, err := f.syncOptionsFromArgs(New(), []string{"/local", "/remote"})
	require.NoError(t, err)
	assert.Equal(t, "/local", opts.LocalPath)
	assert.Equal(t, "/remote", opts.RemotePath)
}
