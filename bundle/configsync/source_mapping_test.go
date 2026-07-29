package configsync

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceSnapshotKeepsPlanAndYAMLOnSameContent(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	directory := t.TempDir()
	path := filepath.Join(directory, "databricks.yml")
	original := "bundle:\n  name: test\nresources:\n  jobs:\n    example:\n      tasks:\n        - task_key: alpha\n          timeout_seconds: 1\n          notebook_task:\n            notebook_path: /Workspace/alpha\n"
	replacedOnDisk := "bundle:\n  name: test\nresources:\n  jobs:\n    example:\n      tasks:\n        - task_key: beta\n          timeout_seconds: 9\n          notebook_task:\n            notebook_path: /Workspace/beta\n"
	require.NoError(t, os.WriteFile(path, []byte(original), 0o644))

	b, err := bundle.Load(ctx, directory)
	require.NoError(t, err)
	mutator.DefaultMutators(ctx, b)
	require.False(t, logdiag.HasError(ctx))
	snapshot, err := CaptureSourceSnapshot(ctx, b)
	require.NoError(t, err)

	// Simulate a watcher replacing the source after snapshot capture but before
	// the remote plan finishes. Routing must stay tied to the captured bytes.
	require.NoError(t, os.WriteFile(path, []byte(replacedOnDisk), 0o644))
	err = snapshot.Validate()
	require.ErrorIs(t, err, ErrSourceChanged)
	fieldChanges, err := ResolveChangesFromSnapshot(ctx, b, Changes{
		"resources.jobs.example": {
			"tasks[task_key='alpha'].timeout_seconds": {
				Operation:   OperationReplace,
				Value:       int64(2),
				configValue: int64(1),
			},
		},
	}, snapshot)
	require.NoError(t, err)
	require.Len(t, fieldChanges, 1)
	assert.Equal(t, original, string(fieldChanges[0].originalFileContent))

	files, err := ApplyChangesToYAML(ctx, b, fieldChanges)
	require.NoError(t, err)
	require.Len(t, files, 1)
	assert.Equal(t, original, files[0].OriginalContent)

	err = SaveFiles(ctx, b, files)
	require.ErrorContains(t, err, "changed while remote changes were being resolved")
	content, readErr := os.ReadFile(path)
	require.NoError(t, readErr)
	assert.Equal(t, replacedOnDisk, string(content))
}

func TestSourceSnapshotReadsFilesAtCapture(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	directory := t.TempDir()
	path := filepath.Join(directory, "databricks.yml")
	loaded := "bundle:\n  name: loaded\n"
	current := "bundle:\n  name: current\n"
	require.NoError(t, os.WriteFile(path, []byte(loaded), 0o644))

	b, err := bundle.Load(ctx, directory)
	require.NoError(t, err)
	mutator.DefaultMutators(ctx, b)
	require.False(t, logdiag.HasError(ctx))
	require.NoError(t, os.WriteFile(path, []byte(current), 0o644))

	snapshot, err := CaptureSourceSnapshot(ctx, b)
	require.NoError(t, err)
	assert.Equal(t, current, string(snapshot.index.files[path].content))
	require.NoError(t, snapshot.Validate())
}

func TestSourceSnapshotDetectsNewIncludeMatch(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	directory := t.TempDir()
	resources := filepath.Join(directory, "resources")
	require.NoError(t, os.MkdirAll(resources, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(directory, "databricks.yml"), []byte("bundle:\n  name: test\ninclude:\n  - resources/*.yml\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(resources, "first.yml"), []byte("resources:\n  jobs:\n    first:\n      name: first\n"), 0o644))

	b, err := bundle.Load(ctx, directory)
	require.NoError(t, err)
	mutator.DefaultMutators(ctx, b)
	require.False(t, logdiag.HasError(ctx))
	snapshot, err := CaptureSourceSnapshot(ctx, b)
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(resources, "second.yml"), []byte("resources:\n  jobs:\n    second:\n      name: second\n"), 0o644))
	err = snapshot.Validate()
	require.ErrorIs(t, err, ErrSourceChanged)
}
