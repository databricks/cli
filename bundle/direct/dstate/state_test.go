package dstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenSaveFinalizeRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(path))

	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{"key": "val"}, nil))
	require.NoError(t, db.Finalize())

	// Re-open and verify persisted data.
	var db2 DeploymentState
	require.NoError(t, db2.Open(path))
	assert.Equal(t, 1, db2.Data.Serial)
	assert.Equal(t, "123", db2.GetResourceID("jobs.my_job"))
}

func TestFinalizeNoOpWhenUnmodified(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(path))
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{}, nil))
	require.NoError(t, db.Finalize())

	// Second Finalize with no changes is a no-op — serial stays at 1.
	require.NoError(t, db.Finalize())

	var db2 DeploymentState
	require.NoError(t, db2.Open(path))
	assert.Equal(t, 1, db2.Data.Serial)
}

func TestFinalizeRetryAfterWriteFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod not effective on Windows")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(path))
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{}, nil))

	// Make directory read-only so WriteFile fails.
	require.NoError(t, os.Chmod(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	err := db.Finalize()
	require.Error(t, err)

	// Restore permissions and retry — must succeed and persist.
	require.NoError(t, os.Chmod(dir, 0o755))
	require.NoError(t, db.Finalize())

	var db2 DeploymentState
	require.NoError(t, db2.Open(path))
	assert.Equal(t, 1, db2.Data.Serial)
	assert.Equal(t, "123", db2.GetResourceID("jobs.my_job"))
}

func TestMigrateStateSetsModified(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	// Write a v0 state file (no state_version field).
	v0State := Database{
		StateVersion: 0,
		CLIVersion:   "0.0.0",
		Lineage:      "test-lineage",
		Serial:       5,
		State:        map[string]ResourceEntry{},
	}
	data, err := json.Marshal(v0State)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))

	// Open triggers migration.
	var db DeploymentState
	require.NoError(t, db.Open(path))
	assert.Equal(t, currentStateVersion, db.Data.StateVersion)

	// Finalize should write the migrated state (modified=true from migration).
	require.NoError(t, db.Finalize())

	// Re-read and verify version was persisted.
	var db2 DeploymentState
	require.NoError(t, db2.Open(path))
	assert.Equal(t, currentStateVersion, db2.Data.StateVersion)
	assert.Equal(t, 6, db2.Data.Serial)
}

func TestPanicOnDoubleOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(path))

	assert.Panics(t, func() {
		_ = db.Open(path)
	})
}

func TestDeleteStateSetsModified(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(path))
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{}, nil))
	require.NoError(t, db.Finalize())

	require.NoError(t, db.DeleteState("jobs.my_job"))
	require.NoError(t, db.Finalize())

	var db2 DeploymentState
	require.NoError(t, db2.Open(path))
	assert.Equal(t, 2, db2.Data.Serial)
	assert.Equal(t, "", db2.GetResourceID("jobs.my_job"))
}
