package bundle

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/vfs"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSyncCommand attaches the sync command to a root command so the root persistent flags resolve.
func newTestSyncCommand(t *testing.T) *cobra.Command {
	syncCmd := newSyncCommand()
	root.New(t.Context()).AddCommand(syncCmd)
	return syncCmd
}

func TestBundleSyncShorthandFlags(t *testing.T) {
	cmd := newTestSyncCommand(t)
	require.NoError(t, cmd.ParseFlags([]string{"-o", "json", "-p", "myprofile"}))
	assert.Equal(t, flags.OutputJSON, root.OutputType(cmd))
	assert.Equal(t, "myprofile", cmd.Flag("profile").Value.String())
}

func TestBundleSyncOutputHandlerOnlyWhenOutputSet(t *testing.T) {
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

	cmd := newTestSyncCommand(t)
	cmd.SetContext(t.Context())
	require.NoError(t, cmd.ParseFlags(nil))
	opts, err := f.syncOptionsFromBundle(cmd, b)
	require.NoError(t, err)
	assert.Nil(t, opts.OutputHandler)

	cmd = newTestSyncCommand(t)
	cmd.SetContext(t.Context())
	require.NoError(t, cmd.ParseFlags([]string{"-o", "json"}))
	opts, err = f.syncOptionsFromBundle(cmd, b)
	require.NoError(t, err)
	assert.NotNil(t, opts.OutputHandler)
}
