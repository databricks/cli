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
	t.Run("fingerprints the matched file set", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("world"), 0o644))

		pattern := "*.txt"
		b := bundleWithFileTrigger(dir, pattern)

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.False(t, diags.HasError())

		fingerprints := b.Config.Resources.JobRuns["my_run"].Lifecycle.TriggersState.OnFileChange
		require.Len(t, fingerprints, 1)
		assert.Equal(t, contentHash(
			"a.txt\x00"+contentHash("hello")+"\x00"+
				"b.txt\x00"+contentHash("world")+"\x00",
		), fingerprints["*.txt"])
	})

	t.Run("rejects an absolute pattern", func(t *testing.T) {
		dir := t.TempDir()
		pattern := "/etc/passwd"
		b := bundleWithFileTrigger(dir, pattern)

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.True(t, diags.HasError())
		require.Equal(t, `lifecycle.triggers.on_file_change: pattern "/etc/passwd" must be relative to the defining YAML file`, diags[0].Summary)
		assert.Empty(t, b.Config.Resources.JobRuns["my_run"].Lifecycle.TriggersState.OnFileChange)
	})

	t.Run("missing pattern is keyed relative to the sync root", func(t *testing.T) {
		parent := t.TempDir()
		bundleDir := filepath.Join(parent, "bundle")
		require.NoError(t, os.Mkdir(bundleDir, 0o755))

		pattern := "../missing.txt"
		b := bundleWithFileTrigger(parent, pattern)
		b.BundleRootPath = bundleDir

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.False(t, diags.HasError())
		assert.Equal(t, map[string]string{"missing.txt": contentHash("")}, b.Config.Resources.JobRuns["my_run"].Lifecycle.TriggersState.OnFileChange)
	})

	t.Run("globs from the bundle root when the sync root is an ancestor", func(t *testing.T) {
		parent := t.TempDir()
		bundleDir := filepath.Join(parent, "bundle")
		require.NoError(t, os.Mkdir(bundleDir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(parent, "shared.txt"), []byte("from-sync-root"), 0o644))

		pattern := "../shared.txt"
		b := bundleWithFileTrigger(parent, pattern)
		b.BundleRootPath = bundleDir

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.False(t, diags.HasError())
		assert.Equal(t,
			contentHash("shared.txt\x00"+contentHash("from-sync-root")+"\x00"),
			b.Config.Resources.JobRuns["my_run"].Lifecycle.TriggersState.OnFileChange["shared.txt"],
		)
	})

	t.Run("hashes through the sync root", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "watched.txt"), []byte("native"), 0o644))
		b := bundleWithFileTrigger(dir, "watched.txt")
		overlay, err := vfs.Overlay(b.SyncRoot, map[string][]byte{"watched.txt": []byte("overlay")})
		require.NoError(t, err)
		b.SyncRoot = overlay

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.False(t, diags.HasError())
		assert.Equal(t,
			contentHash("watched.txt\x00"+contentHash("overlay")+"\x00"),
			b.Config.Resources.JobRuns["my_run"].Lifecycle.TriggersState.OnFileChange["watched.txt"],
		)
	})
}

func bundleWithFileTrigger(syncRoot, pattern string) *bundle.Bundle {
	root := vfs.MustNew(syncRoot)
	return &bundle.Bundle{
		BundleRootPath: syncRoot,
		SyncRootPath:   syncRoot,
		SyncRoot:       root,
		WorktreeRoot:   root,
		Config: config.Root{
			Bundle: config.Bundle{Target: "default"},
			Sync:   config.Sync{Paths: []string{"."}},
			Resources: config.Resources{
				JobRuns: map[string]*resources.JobRun{
					"my_run": {
						Lifecycle: &resources.JobRunLifecycle{
							Triggers: []resources.JobRunTrigger{
								{OnFileChange: &pattern},
							},
							TriggersState: nil,
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
