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
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveJobRunFileTriggers(t *testing.T) {
	t.Run("matches files and fills hashes", func(t *testing.T) {
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

	t.Run("no matches warns and stores empty hash", func(t *testing.T) {
		dir := t.TempDir()
		pattern := "missing.txt"
		b := bundleWithFileTrigger(dir, pattern)

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.False(t, diags.HasError())
		require.Len(t, diags, 1)
		assert.Equal(t, diag.Warning, diags[0].Severity)
		assert.Contains(t, diags[0].Summary, `no files match "missing.txt"`)

		hashes := b.Config.Resources.JobRuns["my_run"].ResolvedFileTriggers
		require.Len(t, hashes, 1)
		assert.Empty(t, hashes["missing.txt"])
	})

	t.Run("no file triggers is a no-op", func(t *testing.T) {
		dir := t.TempDir()
		on := true
		b := &bundle.Bundle{
			SyncRootPath: dir,
			Config: config.Root{
				Resources: config.Resources{
					JobRuns: map[string]*resources.JobRun{
						"my_run": {
							Lifecycle: &resources.JobRunLifecycle{
								Triggers: []resources.JobRunTrigger{
									{OnBundleDeploy: &on},
								},
							},
						},
					},
				},
			},
		}

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		assert.Empty(t, diags)
		assert.Nil(t, b.Config.Resources.JobRuns["my_run"].ResolvedFileTriggers)
	})

	t.Run("multiple patterns merge into one map", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("aaa"), 0o644))
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "subdir"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "subdir", "x.py"), []byte("bbb"), 0o644))

		patA := "a.txt"
		patB := "subdir/*.py"
		b := &bundle.Bundle{
			SyncRootPath: dir,
			Config: config.Root{
				Resources: config.Resources{
					JobRuns: map[string]*resources.JobRun{
						"my_run": {
							Lifecycle: &resources.JobRunLifecycle{
								Triggers: []resources.JobRunTrigger{
									{OnFileChange: &patA},
									{OnFileChange: &patB},
								},
							},
						},
					},
				},
			},
		}

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.False(t, diags.HasError())

		hashes := b.Config.Resources.JobRuns["my_run"].ResolvedFileTriggers
		require.Len(t, hashes, 2)
		assert.Equal(t, contentHash("aaa"), hashes["a.txt"])
		assert.Equal(t, contentHash("bbb"), hashes["subdir/x.py"])
	})

	t.Run("pattern outside sync root is an error", func(t *testing.T) {
		dir := t.TempDir()
		pattern := "../outside.txt"
		b := bundleWithFileTrigger(dir, pattern)

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.True(t, diags.HasError())
		require.Len(t, diags, 1)
		assert.Contains(t, diags[0].Summary, `not under the sync root`)
		assert.Empty(t, b.Config.Resources.JobRuns["my_run"].ResolvedFileTriggers)
	})

	t.Run("directory-only match is an error", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "migrations"), 0o755))
		pattern := "migrations"
		b := bundleWithFileTrigger(dir, pattern)

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.True(t, diags.HasError())
		require.Len(t, diags, 1)
		assert.Contains(t, diags[0].Summary, `matches no regular files`)
		assert.Empty(t, b.Config.Resources.JobRuns["my_run"].ResolvedFileTriggers)
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
	return &bundle.Bundle{
		SyncRootPath: syncRoot,
		Config: config.Root{
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
