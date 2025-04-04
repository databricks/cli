package bundle

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/vfs"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleSyncExcludeFromFlag(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	// Create a bundle
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

	// Create a temporary exclude-from file
	excludeFromPath := filepath.Join(tempDir, "exclude-patterns.txt")
	excludePatterns := []string{
		"*.log",
		"build/",
		"# This is a comment",
		"",
		"temp/*.tmp",
	}
	require.NoError(t, os.WriteFile(excludeFromPath, []byte(strings.Join(excludePatterns, "\n")), 0o644))

	// Set up the flags
	f := syncFlags{
		excludeFrom: excludeFromPath,
	}

	// Test syncOptionsFromBundle
	cmd := &cobra.Command{}
	opts, err := f.syncOptionsFromBundle(cmd, b)
	require.NoError(t, err)

	// Expected patterns (should skip comments and empty lines)
	expected := []string{"*.log", "build/", "temp/*.tmp"}
	assert.ElementsMatch(t, expected, opts.Exclude)

	// Test with both exclude flag and exclude-from flag
	f.exclude = []string{"node_modules/"}
	opts, err = f.syncOptionsFromBundle(cmd, b)
	require.NoError(t, err)

	// Should include both exclude flag and exclude-from patterns
	expected = []string{"node_modules/", "*.log", "build/", "temp/*.tmp"}
	assert.ElementsMatch(t, expected, opts.Exclude)
}
