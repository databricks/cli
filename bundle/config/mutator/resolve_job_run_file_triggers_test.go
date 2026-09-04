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

func TestResolveJobRunFileTriggersHashesThroughSyncRoot(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "watched.txt"), []byte("native"), 0o644))
	root := vfs.MustNew(dir)
	pattern := "watched.txt"
	b := &bundle.Bundle{
		BundleRootPath: dir,
		SyncRootPath:   dir,
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
						},
					},
				},
			},
		},
	}
	overlay, err := vfs.Overlay(b.SyncRoot, map[string][]byte{"watched.txt": []byte("overlay")})
	require.NoError(t, err)
	b.SyncRoot = overlay

	diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
	require.False(t, diags.HasError())
	assert.Equal(t,
		contentHash("watched.txt\x00"+contentHash("overlay")+"\x00"),
		b.Config.Resources.JobRuns["my_run"].Lifecycle.TriggersState.OnFileChange["watched.txt"],
	)
}

// Rooted patterns must be rejected the same way on every host: filepath.IsAbs
// alone only knows the local flavour, so a Windows-rooted pattern used to pass
// validation on POSIX and fail on Windows.
func TestResolveJobRunFileTriggersRejectsRootedPatterns(t *testing.T) {
	for _, pattern := range []string{
		"/abs/watched.txt",
		`C:\watched.txt`,
		"C:/watched.txt",
		"c:watched.txt",
		`\\server\share\watched.txt`,
		`\rooted.txt`,
	} {
		t.Run(pattern, func(t *testing.T) {
			dir := t.TempDir()
			require.NoError(t, os.WriteFile(filepath.Join(dir, "watched.txt"), []byte("native"), 0o644))
			root := vfs.MustNew(dir)
			b := &bundle.Bundle{
				BundleRootPath: dir,
				SyncRootPath:   dir,
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
								},
							},
						},
					},
				},
			}

			diags := bundle.Apply(t.Context(), b, mutator.ResolveJobRunFileTriggers())
			require.True(t, diags.HasError(), "expected %q to be rejected", pattern)
			assert.Contains(t, diags[0].Summary, "must be relative to the defining YAML file")
		})
	}
}

func contentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}
