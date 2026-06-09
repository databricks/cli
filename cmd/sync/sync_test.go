package sync

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/vfs"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestSyncCommand returns the sync command attached to a root command so
// that the root persistent flags (--output, --profile) resolve like in
// production.
func newTestSyncCommand(t *testing.T) *cobra.Command {
	syncCmd := New()
	root.New(t.Context()).AddCommand(syncCmd)
	return syncCmd
}

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
	require.ErrorContains(t, err, "accepts 2 arg(s), received 0")
	_, err = f.syncOptionsFromArgs(New(), []string{"foo"})
	require.ErrorContains(t, err, "accepts 2 arg(s), received 1")
	_, err = f.syncOptionsFromArgs(New(), []string{"foo", "bar", "qux"})
	require.ErrorContains(t, err, "accepts 2 arg(s), received 3")
}

func TestSyncOptionsFromArgs(t *testing.T) {
	local := t.TempDir()
	remote := "/remote"

	f := syncFlags{}
	cmd := newTestSyncCommand(t)
	cmd.SetContext(cmdctx.SetWorkspaceClient(t.Context(), nil))
	opts, err := f.syncOptionsFromArgs(cmd, []string{local, remote})
	require.NoError(t, err)
	assert.Equal(t, local, opts.LocalRoot.Native())
	assert.Equal(t, remote, opts.RemotePath)
}

func TestSyncShorthandFlags(t *testing.T) {
	cmd := newTestSyncCommand(t)
	require.NoError(t, cmd.ParseFlags([]string{"-o", "json", "-p", "myprofile"}))
	assert.Equal(t, flags.OutputJSON, root.OutputType(cmd))
	assert.Equal(t, "myprofile", cmd.Flag("profile").Value.String())
}

func TestReadPatternsFileErrorNamesFlag(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.txt")
	_, err := readPatternsFile("exclude-from", missing)
	assert.ErrorContains(t, err, "failed to read exclude-from file")
	_, err = readPatternsFile("include-from", missing)
	assert.ErrorContains(t, err, "failed to read include-from file")
}

func TestExcludeFromFlag(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()
	local := filepath.Join(tempDir, "local")
	require.NoError(t, os.MkdirAll(local, 0o755))
	remote := "/remote"

	// Create a temporary exclude-from file
	excludeFromPath := filepath.Join(tempDir, "exclude-patterns.txt")
	excludePatterns := []string{
		"*.log",
		"build/",
		"temp/*.tmp",
	}
	require.NoError(t, os.WriteFile(excludeFromPath, []byte(strings.Join(excludePatterns, "\n")), 0o644))

	// Set up the flags
	f := syncFlags{excludeFrom: excludeFromPath}
	cmd := newTestSyncCommand(t)
	cmd.SetContext(cmdctx.SetWorkspaceClient(t.Context(), nil))

	// Test with both exclude flag and exclude-from flag
	f.exclude = []string{"node_modules/"}
	opts, err := f.syncOptionsFromArgs(cmd, []string{local, remote})
	require.NoError(t, err)

	// Should include both exclude flag and exclude-from patterns
	expected := []string{"node_modules/", "*.log", "build/", "temp/*.tmp"}
	assert.ElementsMatch(t, expected, opts.Exclude)
}
