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

	wal, err := openWAL(statePath)
	require.NoError(t, err)

	err = wal.writeHeader("test-lineage", 1)
	require.NoError(t, err)

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

	err = wal.writeEntry("resources.jobs.old_job", nil)
	require.NoError(t, err)

	err = wal.close()
	require.NoError(t, err)

	ctx := context.Background()
	header, entries, err := readWAL(ctx, statePath)
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

	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("test-lineage", 1)
	require.NoError(t, err)

	_, err = os.Stat(walFilePath)
	require.NoError(t, err)

	err = wal.truncate()
	require.NoError(t, err)

	_, err = os.Stat(walFilePath)
	assert.True(t, os.IsNotExist(err))
}

func TestRecoverFromWAL_NoWAL(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	db := NewDatabase("", 0)
	recovered, err := recoverFromWAL(ctx, statePath, &db)
	require.NoError(t, err)
	assert.False(t, recovered)
}

func TestRecoverFromWAL_ValidWAL(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

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

	db := NewDatabase("", 0)

	recovered, err := recoverFromWAL(ctx, statePath, &db)
	require.NoError(t, err)
	assert.True(t, recovered)

	assert.Equal(t, "test-lineage", db.Lineage)
	require.Contains(t, db.State, "resources.jobs.job1")
	assert.Equal(t, "12345", db.State["resources.jobs.job1"].ID)
}

func TestRecoverFromWAL_StaleWAL(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")
	walFilePath := walPath(statePath)

	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("test-lineage", 1)
	require.NoError(t, err)
	err = wal.close()
	require.NoError(t, err)

	db := NewDatabase("test-lineage", 2) // serial 2 makes WAL stale

	recovered, err := recoverFromWAL(ctx, statePath, &db)
	require.NoError(t, err)
	assert.False(t, recovered)

	_, err = os.Stat(walFilePath)
	assert.True(t, os.IsNotExist(err))
}

func TestRecoverFromWAL_FutureWAL(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("test-lineage", 5)
	require.NoError(t, err)
	err = wal.close()
	require.NoError(t, err)

	db := NewDatabase("test-lineage", 0)

	_, err = recoverFromWAL(ctx, statePath, &db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ahead of expected")
}

func TestRecoverFromWAL_LineageMismatch(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("lineage-A", 1)
	require.NoError(t, err)
	err = wal.close()
	require.NoError(t, err)

	db := NewDatabase("lineage-B", 0)

	_, err = recoverFromWAL(ctx, statePath, &db)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lineage")
}

func TestRecoverFromWAL_DeleteOperation(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

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

	err = wal.writeEntry("resources.jobs.job1", nil)
	require.NoError(t, err)

	err = wal.close()
	require.NoError(t, err)

	db := NewDatabase("", 0)

	recovered, err := recoverFromWAL(ctx, statePath, &db)
	require.NoError(t, err)
	assert.True(t, recovered)

	assert.NotContains(t, db.State, "resources.jobs.job1")
}

func TestDeploymentState_WALIntegration(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")
	walFilePath := walPath(statePath)

	var db DeploymentState
	err := db.Open(ctx, statePath)
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job1", "12345", map[string]string{"name": "job1"}, nil)
	require.NoError(t, err)

	_, err = os.Stat(walFilePath)
	require.NoError(t, err)

	header, entries, err := readWAL(ctx, statePath)
	require.NoError(t, err)
	assert.Equal(t, 1, header.Serial)
	require.Len(t, entries, 1)
	assert.Equal(t, "resources.jobs.job1", entries[0].K)
	assert.Equal(t, "12345", entries[0].V.ID)

	err = db.Finalize()
	require.NoError(t, err)

	_, err = os.Stat(walFilePath)
	assert.True(t, os.IsNotExist(err))

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

	initialDB := NewDatabase("test-lineage", 5)
	initialDB.State["resources.jobs.existing"] = ResourceEntry{
		ID:    "existing-id",
		State: json.RawMessage(`{"name":"existing"}`),
	}
	data, err := json.Marshal(initialDB)
	require.NoError(t, err)
	err = os.WriteFile(statePath, data, 0o600)
	require.NoError(t, err)

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

	var db DeploymentState
	err = db.Open(ctx, statePath)
	require.NoError(t, err)

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

	err = db.SaveState("resources.jobs.job1", "12345", map[string]string{"name": "job1"}, nil)
	require.NoError(t, err)

	err = db.DeleteState("resources.jobs.job1")
	require.NoError(t, err)

	_, entries, err := readWAL(ctx, statePath)
	require.NoError(t, err)

	require.Len(t, entries, 2)
	assert.Equal(t, "resources.jobs.job1", entries[1].K)
	assert.Nil(t, entries[1].V)

	err = db.Finalize()
	require.NoError(t, err)

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

	_, entries, err := readWAL(ctx, statePath)
	require.NoError(t, err)

	require.Len(t, entries, 1)
	require.NotNil(t, entries[0].V)
	require.Len(t, entries[0].V.DependsOn, 1)
	assert.Equal(t, "resources.clusters.cluster1", entries[0].V.DependsOn[0].Node)
}

func TestRecoverFromWAL_CorruptedMiddleLine(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")
	walFilePath := walPath(statePath)

	content := `{"lineage":"test","serial":1}
{"k":"resources.jobs.job1","v":{"__id__":"12345","state":{}}}
not valid json
{"k":"resources.jobs.job2","v":{"__id__":"67890","state":{}}}
`
	err := os.WriteFile(walFilePath, []byte(content), 0o600)
	require.NoError(t, err)

	db := NewDatabase("", 0)
	recovered, err := recoverFromWAL(ctx, statePath, &db)
	require.NoError(t, err)
	assert.False(t, recovered)
	assert.Empty(t, db.State)

	_, err = os.Stat(walFilePath)
	assert.True(t, os.IsNotExist(err))
}

func TestRecoverFromWAL_CorruptedLastLine(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")
	walFilePath := walPath(statePath)

	content := `{"lineage":"test","serial":1}
{"k":"resources.jobs.job1","v":{"__id__":"12345","state":{}}}
{"k":"resources.jobs.job2","v":{"__id__":"67890","state":{}}}
not valid json
`
	err := os.WriteFile(walFilePath, []byte(content), 0o600)
	require.NoError(t, err)

	db := NewDatabase("", 0)
	recovered, err := recoverFromWAL(ctx, statePath, &db)
	require.NoError(t, err)
	assert.True(t, recovered)

	assert.Contains(t, db.State, "resources.jobs.job1")
	assert.Contains(t, db.State, "resources.jobs.job2")
	assert.Equal(t, "12345", db.State["resources.jobs.job1"].ID)
	assert.Equal(t, "67890", db.State["resources.jobs.job2"].ID)
}

func TestDeploymentState_RecoveredFromWALFlag(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	initialDB := NewDatabase("test-lineage", 0)
	data, err := json.Marshal(initialDB)
	require.NoError(t, err)
	err = os.WriteFile(statePath, data, 0o600)
	require.NoError(t, err)

	wal, err := openWAL(statePath)
	require.NoError(t, err)
	err = wal.writeHeader("test-lineage", 1)
	require.NoError(t, err)
	err = wal.writeEntry("resources.jobs.job1", &ResourceEntry{ID: "123", State: json.RawMessage(`{}`)})
	require.NoError(t, err)
	err = wal.close()
	require.NoError(t, err)

	var db DeploymentState
	err = db.Open(ctx, statePath)
	require.NoError(t, err)

	assert.True(t, db.RecoveredFromWAL())
}

func TestRecoverFromWAL_LineageAdoption(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")
	walFilePath := walPath(statePath)

	content := `{"lineage":"adopted-lineage","serial":1}
{"k":"resources.jobs.job1","v":{"__id__":"12345","state":{}}}
`
	err := os.WriteFile(walFilePath, []byte(content), 0o600)
	require.NoError(t, err)

	db := NewDatabase("", 0) // empty lineage
	recovered, err := recoverFromWAL(ctx, statePath, &db)
	require.NoError(t, err)
	assert.True(t, recovered)
	assert.Equal(t, "adopted-lineage", db.Lineage)
}

func TestReadWAL_EmptyFile(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")
	walFilePath := walPath(statePath)

	err := os.WriteFile(walFilePath, []byte(""), 0o600)
	require.NoError(t, err)

	_, _, err = readWAL(ctx, statePath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestDeploymentState_MultipleOperationsSameKey(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	statePath := filepath.Join(dir, "resources.json")

	var db DeploymentState
	err := db.Open(ctx, statePath)
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job1", "111", map[string]string{"v": "1"}, nil)
	require.NoError(t, err)

	err = db.DeleteState("resources.jobs.job1")
	require.NoError(t, err)

	err = db.SaveState("resources.jobs.job1", "222", map[string]string{"v": "2"}, nil)
	require.NoError(t, err)

	_, entries, err := readWAL(ctx, statePath)
	require.NoError(t, err)
	require.Len(t, entries, 3)
	assert.Equal(t, "111", entries[0].V.ID)
	assert.Nil(t, entries[1].V)
	assert.Equal(t, "222", entries[2].V.ID)

	err = db.Finalize()
	require.NoError(t, err)

	entry, ok := db.GetResourceEntry("resources.jobs.job1")
	require.True(t, ok)
	assert.Equal(t, "222", entry.ID)
}
