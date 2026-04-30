package dstate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenCloseRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))

	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{"key": "val"}, nil))
	require.NoError(t, db.Finalize(t.Context()))

	// Re-open and verify persisted data.
	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(false), WithWrite(false)))
	assert.Equal(t, 1, db2.Data.Serial)
	assert.Equal(t, "123", db2.GetResourceID("jobs.my_job"))
	require.NoError(t, db2.Finalize(t.Context()))
}

func TestCloseWithNoEntriesDoesNotWriteStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	require.NoError(t, db.Finalize(t.Context()))

	_, err := os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestPanicOnDoubleOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))

	assert.Panics(t, func() {
		_ = db.Open(t.Context(), path, WithRecovery(true), WithWrite(true))
	})
	db.Finalize(t.Context())
}

func TestDeleteState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{}, nil))
	require.NoError(t, db.Finalize(t.Context()))

	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	require.NoError(t, db2.DeleteState("jobs.my_job"))
	require.NoError(t, db2.Finalize(t.Context()))

	var db3 DeploymentState
	require.NoError(t, db3.Open(t.Context(), path, WithRecovery(false), WithWrite(false)))
	assert.Equal(t, 2, db3.Data.Serial)
	assert.Equal(t, "", db3.GetResourceID("jobs.my_job"))
	require.NoError(t, db3.Finalize(t.Context()))
}
