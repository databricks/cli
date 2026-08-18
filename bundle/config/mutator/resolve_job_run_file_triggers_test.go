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
	t.Run("matches files and fills fingerprints", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "b.txt"), []byte("world"), 0o644))

		pattern := "*.txt"
		b := bundleWithFileTrigger(dir, pattern)

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.False(t, diags.HasError())

		fps := b.Config.Resources.JobRuns["my_run"].ResolvedFileTriggers
		require.Len(t, fps, 2)

		assertFingerprint(t, fps["a.txt"], "hello")
		assertFingerprint(t, fps["b.txt"], "world")
	})

	t.Run("no matches warns and stores sentinel", func(t *testing.T) {
		dir := t.TempDir()
		pattern := "missing.txt"
		b := bundleWithFileTrigger(dir, pattern)

		diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
		require.False(t, diags.HasError())
		require.Len(t, diags, 1)
		assert.Equal(t, diag.Warning, diags[0].Severity)
		assert.Contains(t, diags[0].Summary, `no files match "missing.txt"`)

		fps := b.Config.Resources.JobRuns["my_run"].ResolvedFileTriggers
		require.Len(t, fps, 1)
		fp := fps["missing.txt"]
		assert.Empty(t, fp.Hash)
		assert.Equal(t, int64(-1), fp.Size)
		assert.Zero(t, fp.MtimeNs)
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

		fps := b.Config.Resources.JobRuns["my_run"].ResolvedFileTriggers
		require.Len(t, fps, 2)
		assertFingerprint(t, fps["a.txt"], "aaa")
		assertFingerprint(t, fps["subdir/x.py"], "bbb")
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

func assertFingerprint(t *testing.T, fp resources.JobRunFileFingerprint, content string) {
	t.Helper()
	sum := sha256.Sum256([]byte(content))
	assert.Equal(t, hex.EncodeToString(sum[:]), fp.Hash)
	assert.Equal(t, int64(len(content)), fp.Size)
	assert.NotZero(t, fp.MtimeNs)
}
