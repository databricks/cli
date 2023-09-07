package sync

import (
	"context"
	"flag"
	"os"
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

			Workspace: config.Workspace{
				FilesPath: "/Users/jane@doe.com/path",
			},
		},
	}

	f := syncFlags{}
	opts, err := f.syncOptionsFromBundle(New(), b)
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

func TestSyncOptonsReturnsErrorFromArgs(t *testing.T) {
	f := syncFlags{}
	_, err := f.syncOptions(New(), []string{"/local"}, bundle.Seq())
	require.ErrorIs(t, err, flag.ErrHelp)
}

func TestSyncOptionsReturnsFromBundle(t *testing.T) {
	tempDir := t.TempDir()
	file, err := os.Create(filepath.Join(tempDir, "bundle.yaml"))
	t.Cleanup(func() {
		err := file.Close()
		assert.NoError(t, err)
	})

	assert.NoError(t, err)
	data := `
bundle:
  name: test
workspace:
  file_path: /Users/jane@doe.com/path
`
	_, err = file.WriteString(data)
	assert.NoError(t, err)

	t.Setenv("BUNDLE_ROOT", tempDir)

	cmd := New()
	cmd.SetContext(context.Background())

	f := syncFlags{}
	opts, err := f.syncOptions(cmd, []string{"/local"}, bundle.Seq())
	assert.NoError(t, err)
	assert.Equal(t, opts.RemotePath, "/Users/jane@doe.com/path")
	assert.Equal(t, opts.LocalPath, tempDir)
}
