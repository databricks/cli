package bundle

import (
	"io"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Flags that bundle sync defines with the names or shorthands of the root's
// --output and --profile flags would shadow them.
func TestBundleSyncDoesNotShadowRootFlags(t *testing.T) {
	cmd := newSyncCommand()
	for _, name := range []string{"output", "profile"} {
		assert.Nil(t, cmd.LocalFlags().Lookup(name), name)
	}
	for _, shorthand := range []string{"o", "p"} {
		assert.Nil(t, cmd.LocalFlags().ShorthandLookup(shorthand), shorthand)
	}
}

func TestBundleSyncInstallsOutputHandler(t *testing.T) {
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

	for _, output := range []flags.Output{flags.OutputText, flags.OutputJSON} {
		opts, err := f.syncOptionsFromBundle(t.Context(), output, io.Discard, b)
		require.NoError(t, err)
		assert.NotNil(t, opts.OutputHandler, output)
	}
}
