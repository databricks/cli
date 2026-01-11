package dstate

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWALPath(t *testing.T) {
	assert.Equal(t, "/path/to/state.json.wal", walPath("/path/to/state.json"))
}

func TestWALWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	// Open WAL for writing
	wal, err := openWAL(statePath)
	require.NoError(t, err)

	// Write header
	err = wal.writeHeader("test-lineage", 1)
	require.NoError(t, err)

	// Write entries
	entry1 := &ResourceEntry{
		ID:    "12345",
		State: json.RawMessage(`{"name":"job1"}`),
	}
	err = wal.writeEntry("resources.jobs.job1", entry1)
	require.NoError(t, err)

	entry2 := &ResourceEntry{
		ID:    "67890",
		State: json.RawMessage(`{"name":"job2"}`),
	}
	err = wal.writeEntry("resources.jobs.job2", entry2)
	require.NoError(t, err)

	// Write a delete entry (nil value)
	err = wal.writeEntry("resources.jobs.old_job", nil)
	require.NoError(t, err)

	err = wal.close()
	require.NoError(t, err)

	// Read WAL back
	header, entries, err := readWAL(statePath)
	require.NoError(t, err)

	assert.Equal(t, "test-lineage", header.Lineage)
	assert.Equal(t, 1, header.Serial)

	require.Len(t, entries, 3)

	assert.Equal(t, "resources.jobs.job1", entries[0].K)
	require.NotNil(t, entries[0].V)
	assert.Equal(t, "12345", entries[0].V.ID)

	assert.Equal(t, "resources.jobs.job2", entries[1].K)
	require.NotNil(t, entries[1].V)
	assert.Equal(t, "67890", entries[1].V.ID)

	assert.Equal(t, "resources.jobs.old_job", entries[2].K)
	assert.Nil(t, entries[2].V)
}

func TestWALTruncate(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")
	walFilePath := walPath(statePath)

	// Create WAL file
	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("test-lineage", 1)
	require.NoError(t, err)

	// Verify file exists
	_, err = os.Stat(walFilePath)
	require.NoError(t, err)

	// Truncate
	err = wal.truncate()
	require.NoError(t, err)

	// Verify file is removed
	_, err = os.Stat(walFilePath)
	assert.True(t, os.IsNotExist(err))
}

func TestRecoverFromWAL_NoWAL(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	db := NewDatabase("", 0)
	recovered, err := recoverFromWAL(statePath, &db)
	require.NoError(t, err)
	assert.False(t, recovered)
}

func TestRecoverFromWAL_ValidWAL(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	// Create WAL with serial = 1 (expecting state serial 0 + 1)
	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("test-lineage", 1)
	require.NoError(t, err)

	entry := &ResourceEntry{
		ID:    "12345",
		State: json.RawMessage(`{"name":"job1"}`),
	}
	err = wal.writeEntry("resources.jobs.job1", entry)
	require.NoError(t, err)
	err = wal.close()
	require.NoError(t, err)

	// Create database with serial 0
	db := NewDatabase("", 0)

	// Recover
	recovered, err := recoverFromWAL(statePath, &db)
	require.NoError(t, err)
	assert.True(t, recovered)

	// Verify state was recovered
	assert.Equal(t, "test-lineage", db.Lineage)
	require.Contains(t, db.State, "resources.jobs.job1")
	assert.Equal(t, "12345", db.State["resources.jobs.job1"].ID)
}

func TestRecoverFromWAL_StaleWAL(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")
	walFilePath := walPath(statePath)

	// Create WAL with serial = 1
	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("test-lineage", 1)
	require.NoError(t, err)
	err = wal.close()
	require.NoError(t, err)

	// Create database with serial 2 (WAL is stale)
	db := NewDatabase("test-lineage", 2)

	// Recover - should skip and delete WAL
	recovered, err := recoverFromWAL(statePath, &db)
	require.NoError(t, err)
	assert.False(t, recovered)

	// WAL should be deleted
	_, err = os.Stat(walFilePath)
	assert.True(t, os.IsNotExist(err))
}

func TestRecoverFromWAL_FutureWAL(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	// Create WAL with serial = 5
	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("test-lineage", 5)
	require.NoError(t, err)
	err = wal.close()
	require.NoError(t, err)

	// Create database with serial 0 (WAL is from future - corrupted state)
	db := NewDatabase("test-lineage", 0)

	// Recover - should fail
	_, err = recoverFromWAL(statePath, &db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "WAL serial (5) is ahead of expected (1)")
}

func TestRecoverFromWAL_LineageMismatch(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	// Create WAL with lineage A
	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("lineage-A", 1)
	require.NoError(t, err)
	err = wal.close()
	require.NoError(t, err)

	// Create database with lineage B
	db := NewDatabase("lineage-B", 0)

	// Recover - should fail
	_, err = recoverFromWAL(statePath, &db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lineage")
}

func TestRecoverFromWAL_DeleteOperation(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	// Create WAL with delete operation
	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("test-lineage", 1)
	require.NoError(t, err)

	// Add an entry
	entry := &ResourceEntry{
		ID:    "12345",
		State: json.RawMessage(`{"name":"job1"}`),
	}
	err = wal.writeEntry("resources.jobs.job1", entry)
	require.NoError(t, err)

	// Delete the entry
	err = wal.writeEntry("resources.jobs.job1", nil)
	require.NoError(t, err)

	err = wal.close()
	require.NoError(t, err)

	// Create database
	db := NewDatabase("", 0)

	// Recover
	recovered, err := recoverFromWAL(statePath, &db)
	require.NoError(t, err)
	assert.True(t, recovered)

	// Entry should NOT be present (deleted)
	assert.NotContains(t, db.State, "resources.jobs.job1")
}

func TestDeploymentState_WALIntegration(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")
	walFilePath := walPath(statePath)

	// Create deployment state
	var db DeploymentState
	err := db.Open(ctx, statePath)
	require.NoError(t, err)

	// Save some state
	err = db.SaveState("resources.jobs.job1", "12345", map[string]string{"name": "job1"}, nil)
	require.NoError(t, err)

	// WAL should exist
	_, err = os.Stat(walFilePath)
	require.NoError(t, err)

	// Read WAL to verify content
	header, entries, err := readWAL(statePath)
	require.NoError(t, err)
	assert.Equal(t, 1, header.Serial) // serial + 1
	require.Len(t, entries, 1)
	assert.Equal(t, "resources.jobs.job1", entries[0].K)
	assert.Equal(t, "12345", entries[0].V.ID)

	// Finalize
	err = db.Finalize()
	require.NoError(t, err)

	// WAL should be deleted
	_, err = os.Stat(walFilePath)
	assert.True(t, os.IsNotExist(err))

	// State file should exist with correct serial
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var savedDB Database
	err = json.Unmarshal(data, &savedDB)
	require.NoError(t, err)
	assert.Equal(t, 1, savedDB.Serial)
	assert.Contains(t, savedDB.State, "resources.jobs.job1")
}

func TestDeploymentState_WALRecoveryOnOpen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	// Create initial state file
	initialDB := NewDatabase("test-lineage", 5)
	initialDB.State["resources.jobs.existing"] = ResourceEntry{
		ID:    "existing-id",
		State: json.RawMessage(`{"name":"existing"}`),
	}
	data, err := json.Marshal(initialDB)
	require.NoError(t, err)
	err = os.WriteFile(statePath, data, 0o600)
	require.NoError(t, err)

	// Create WAL with serial 6 (5 + 1)
	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("test-lineage", 6)
	require.NoError(t, err)
	entry := &ResourceEntry{
		ID:    "new-id",
		State: json.RawMessage(`{"name":"new"}`),
	}
	err = wal.writeEntry("resources.jobs.new", entry)
	require.NoError(t, err)
	err = wal.close()
	require.NoError(t, err)

	// Open should recover from WAL
	var db DeploymentState
	err = db.Open(ctx, statePath)
	require.NoError(t, err)

	// Both existing and new resources should be present
	assert.Contains(t, db.Data.State, "resources.jobs.existing")
	assert.Contains(t, db.Data.State, "resources.jobs.new")
	assert.Equal(t, "new-id", db.Data.State["resources.jobs.new"].ID)
}

func TestDeploymentState_DeleteStateWritesWAL(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	var db DeploymentState
	err := db.Open(ctx, statePath)
	require.NoError(t, err)

	// Add a resource
	err = db.SaveState("resources.jobs.job1", "12345", map[string]string{"name": "job1"}, nil)
	require.NoError(t, err)

	// Delete the resource
	err = db.DeleteState("resources.jobs.job1")
	require.NoError(t, err)

	// Read WAL to verify delete entry
	_, entries, err := readWAL(statePath)
	require.NoError(t, err)

	require.Len(t, entries, 2)
	assert.Equal(t, "resources.jobs.job1", entries[1].K)
	assert.Nil(t, entries[1].V) // nil means delete

	// Finalize
	err = db.Finalize()
	require.NoError(t, err)

	// State file should NOT contain the deleted resource
	data, err := os.ReadFile(statePath)
	require.NoError(t, err)
	var savedDB Database
	err = json.Unmarshal(data, &savedDB)
	require.NoError(t, err)
	assert.NotContains(t, savedDB.State, "resources.jobs.job1")
}

func TestDeploymentState_WALWithDependsOn(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	var db DeploymentState
	err := db.Open(ctx, statePath)
	require.NoError(t, err)

	dependsOn := []deployplan.DependsOnEntry{
		{Node: "resources.clusters.cluster1", Label: "${resources.clusters.cluster1.id}"},
	}

	err = db.SaveState("resources.jobs.job1", "12345", map[string]string{"name": "job1"}, dependsOn)
	require.NoError(t, err)

	// Read WAL
	_, entries, err := readWAL(statePath)
	require.NoError(t, err)

	require.Len(t, entries, 1)
	require.NotNil(t, entries[0].V)
	require.Len(t, entries[0].V.DependsOn, 1)
	assert.Equal(t, "resources.clusters.cluster1", entries[0].V.DependsOn[0].Node)
}

func TestRecoverFromWAL_CorruptedLine(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")
	walFilePath := walPath(statePath)

	// Manually write WAL with corrupted line
	content := `{"lineage":"test","serial":1}
{"k":"resources.jobs.job1","v":{"__id__":"12345","state":{}}}
not valid json
{"k":"resources.jobs.job2","v":{"__id__":"67890","state":{}}}
`
	err := os.WriteFile(walFilePath, []byte(content), 0o600)
	require.NoError(t, err)

	db := NewDatabase("", 0)
	recovered, err := recoverFromWAL(statePath, &db)
	require.NoError(t, err)
	assert.True(t, recovered)

	// Should have recovered job1 and job2, skipping corrupted line
	assert.Contains(t, db.State, "resources.jobs.job1")
	assert.Contains(t, db.State, "resources.jobs.job2")
}

