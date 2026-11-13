package sync

import (
	"io"
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
	opts, err := f.syncOptionsFromBundle(t.Context(), []string{}, b)
	require.NoError(t, err)
	assert.Equal(t, tempDir, opts.LocalRoot.Native())
	assert.Equal(t, "/Users/jane@doe.com/path", opts.RemotePath)
	assert.Equal(t, filepath.Join(tempDir, ".databricks", "bundle", "default"), opts.SnapshotBasePath)
	assert.NotNil(t, opts.WorkspaceClient)
}

func TestSyncOptionsFromArgs(t *testing.T) {
	local := t.TempDir()
	remote := "/remote"

	f := syncFlags{}
	ctx := cmdctx.SetWorkspaceClient(t.Context(), nil)
	opts, err := f.syncOptionsFromArgs(ctx, flags.OutputText, io.Discard, []string{local, remote})
	require.NoError(t, err)
	assert.Equal(t, local, opts.LocalRoot.Native())
	assert.Equal(t, remote, opts.RemotePath)
}

func TestSyncOptionsFromArgsRequiresSrcAndDst(t *testing.T) {
	f := syncFlags{}
	_, err := f.syncOptionsFromArgs(t.Context(), flags.OutputText, io.Discard, []string{"src"})
	assert.EqualError(t, err, "expected SRC and DST, got 1 arguments")
}

// Flags that sync defines with the names or shorthands of the root's
// --output and --profile flags would shadow them.
func TestSyncDoesNotShadowRootFlags(t *testing.T) {
	cmd := root.NewCommand(t.Context(), Spec)
	for _, name := range []string{"output", "profile"} {
		assert.Nil(t, cmd.LocalFlags().Lookup(name), name)
	}
	for _, shorthand := range []string{"o", "p"} {
		assert.Nil(t, cmd.LocalFlags().ShorthandLookup(shorthand), shorthand)
	}
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
	ctx := cmdctx.SetWorkspaceClient(t.Context(), nil)

	// Test with both exclude flag and exclude-from flag
	f.exclude = []string{"node_modules/"}
	opts, err := f.syncOptionsFromArgs(ctx, flags.OutputText, io.Discard, []string{local, remote})
	require.NoError(t, err)

	// Should include both exclude flag and exclude-from patterns
	expected := []string{"node_modules/", "*.log", "build/", "temp/*.tmp"}
	assert.ElementsMatch(t, expected, opts.Exclude)
}
