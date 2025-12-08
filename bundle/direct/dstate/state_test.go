package dstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWALSaveAndReplay(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	db := &DeploymentState{}
	err := db.Open(statePath)
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job1", "123", map[string]any{"name": "job1"})
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job2", "456", map[string]any{"name": "job2"})
	require.NoError(t, err)

	// Verify WAL exists in file
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var loaded Database
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Len(t, loaded.WAL, 2)

	// Simulate crash: open a new state (WAL replays on open)
	db2 := &DeploymentState{}
	err = db2.Open(statePath)
	require.NoError(t, err)

	entry1, ok := db2.GetResourceEntry("resources.jobs.job1")
	assert.True(t, ok)
	assert.Equal(t, "123", entry1.ID)

	entry2, ok := db2.GetResourceEntry("resources.jobs.job2")
	assert.True(t, ok)
	assert.Equal(t, "456", entry2.ID)
}

func TestWALDeleteAndReplay(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	db := &DeploymentState{}
	err := db.Open(statePath)
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job1", "123", map[string]any{"name": "job1"})
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job2", "456", map[string]any{"name": "job2"})
	require.NoError(t, err)

	err = db.DeleteState("resources.jobs.job1")
	require.NoError(t, err)

	// Simulate crash and reopen
	db2 := &DeploymentState{}
	err = db2.Open(statePath)
	require.NoError(t, err)

	_, ok := db2.GetResourceEntry("resources.jobs.job1")
	assert.False(t, ok, "job1 should be deleted")

	entry2, ok := db2.GetResourceEntry("resources.jobs.job2")
	assert.True(t, ok)
	assert.Equal(t, "456", entry2.ID)
}

func TestWALFinalizeClearsWAL(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	db := &DeploymentState{}
	err := db.Open(statePath)
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job1", "123", map[string]any{"name": "job1"})
	require.NoError(t, err)

	// Verify WAL has entries before finalize
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var before Database
	require.NoError(t, json.Unmarshal(data, &before))
	assert.NotEmpty(t, before.WAL)

	err = db.Finalize()
	require.NoError(t, err)

	// Verify WAL is cleared after finalize
	data, err = os.ReadFile(statePath)
	require.NoError(t, err)
	var after Database
	require.NoError(t, json.Unmarshal(data, &after))
	assert.Empty(t, after.WAL)

	// State still has the entry
	db2 := &DeploymentState{}
	err = db2.Open(statePath)
	require.NoError(t, err)

	entry, ok := db2.GetResourceEntry("resources.jobs.job1")
	assert.True(t, ok)
	assert.Equal(t, "123", entry.ID)
}

func TestWALWithExistingState(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	// Create state with one entry and finalize
	db := &DeploymentState{}
	err := db.Open(statePath)
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job1", "123", map[string]any{"name": "job1"})
	require.NoError(t, err)

	err = db.Finalize()
	require.NoError(t, err)

	// Open again and add more entries without finalizing
	db2 := &DeploymentState{}
	err = db2.Open(statePath)
	require.NoError(t, err)

	err = db2.SaveState("resources.jobs.job2", "456", map[string]any{"name": "job2"})
	require.NoError(t, err)

	// Simulate crash and reopen
	db3 := &DeploymentState{}
	err = db3.Open(statePath)
	require.NoError(t, err)

	entry1, ok := db3.GetResourceEntry("resources.jobs.job1")
	assert.True(t, ok)
	assert.Equal(t, "123", entry1.ID)

	entry2, ok := db3.GetResourceEntry("resources.jobs.job2")
	assert.True(t, ok)
	assert.Equal(t, "456", entry2.ID)
}

func TestWALUpdateExistingEntry(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	db := &DeploymentState{}
	err := db.Open(statePath)
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job1", "123", map[string]any{"name": "job1"})
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job1", "789", map[string]any{"name": "job1-updated"})
	require.NoError(t, err)

	// Simulate crash and reopen
	db2 := &DeploymentState{}
	err = db2.Open(statePath)
	require.NoError(t, err)

	entry, ok := db2.GetResourceEntry("resources.jobs.job1")
	assert.True(t, ok)
	assert.Equal(t, "789", entry.ID)
}

func TestNoStateFile(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := filepath.Join(tmpDir, "state.json")

	db := &DeploymentState{}
	err := db.Open(statePath)
	require.NoError(t, err)

	err = db.Finalize()
	require.NoError(t, err)

	db2 := &DeploymentState{}
	err = db2.Open(statePath)
	require.NoError(t, err)

	assert.NotEmpty(t, db2.Data.Lineage)
}
