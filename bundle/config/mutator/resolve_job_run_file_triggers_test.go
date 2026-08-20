package mutator_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveJobRunFileTriggers(t *testing.T) {
	t.Run("hashes file contents with sha256", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("world"), 0o644))

		pattern := "*.txt"
		b := bundleWithFileTrigger(dir, pattern)

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.False(t, diags.HasError())

		hashes := b.Config.Resources.JobRuns["my_run"].ResolvedFileTriggers
		require.Len(t, hashes, 2)
		assert.Equal(t, contentHash("hello"), hashes["a.txt"])
		assert.Equal(t, contentHash("world"), hashes["b.txt"])
	})

	t.Run("trims pattern whitespace", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "seed.txt"), []byte("v1"), 0o644))
		pattern := "  seed.txt  "
		b := bundleWithFileTrigger(dir, pattern)

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.False(t, diags.HasError())
		assert.Equal(t, contentHash("v1"), b.Config.Resources.JobRuns["my_run"].ResolvedFileTriggers["seed.txt"])
	})
}

func bundleWithFileTrigger(syncRoot, pattern string) *bundle.Bundle {
	root := vfs.MustNew(syncRoot)
	return &bundle.Bundle{
		SyncRootPath: syncRoot,
		SyncRoot:     root,
		WorktreeRoot: root,
		Config: config.Root{
			Sync: config.Sync{Paths: []string{"."}},
			Resources: config.Resources{
				JobRuns: map[string]*resources.JobRun{
					"my_run": {
						Lifecycle: &resources.JobRunLifecycle{
							Triggers: []resources.JobRunTrigger{
								{OnFileChange: &pattern},
							},
						},
					},
				},
			},
		},
	}
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
