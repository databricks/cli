package sync

import (
	"context"
	"flag"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncOptionsFromBundle(t *testing.T) {
	tempDir := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: tempDir,
		BundleRoot:     vfs.MustNew(tempDir),
		SyncRootPath:   tempDir,
		SyncRoot:       vfs.MustNew(tempDir),
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "default",
			},

			Workspace: config.Workspace{
				FilePath: "/Users/jane@doe.com/path",
			},
		},
	}

	f := syncFlags{}
	opts, err := f.syncOptionsFromBundle(New(), []string{}, b)
	require.NoError(t, err)
	assert.Equal(t, tempDir, opts.LocalRoot.Native())
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
	local := t.TempDir()
	remote := "/remote"

	f := syncFlags{}
	cmd := New()
	cmd.SetContext(root.SetWorkspaceClient(context.Background(), nil))
	opts, err := f.syncOptionsFromArgs(cmd, []string{local, remote})
	require.NoError(t, err)
	assert.Equal(t, local, opts.LocalRoot.Native())
	assert.Equal(t, remote, opts.RemotePath)
}
