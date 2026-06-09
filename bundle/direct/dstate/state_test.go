package dstate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustFinalize(t *testing.T, db *DeploymentState) {
	t.Helper()
	_, err := db.Finalize(t.Context())
	require.NoError(t, err)
}

func TestOpenSaveFinalizeRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))

	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{"key": "val"}, nil))
	mustFinalize(t, &db)

	// Re-open and verify persisted data.
	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(false), WithWrite(false), nil))
	assert.Equal(t, 1, db2.Data.Serial)
	assert.Equal(t, "123", db2.GetResourceID("jobs.my_job"))
	mustFinalize(t, &db2)
}

func TestFinalizeWithNoEntriesDoesNotWriteStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))
	mustFinalize(t, &db)

	_, err := os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestPanicOnDoubleOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))

	assert.Panics(t, func() {
		_ = db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil)
	})
	mustFinalize(t, &db)
}

func TestDeleteState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{}, nil))
	mustFinalize(t, &db)

	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))
	require.NoError(t, db2.DeleteState("jobs.my_job"))
	mustFinalize(t, &db2)

	var db3 DeploymentState
	require.NoError(t, db3.Open(t.Context(), path, WithRecovery(false), WithWrite(false), nil))
	assert.Equal(t, 2, db3.Data.Serial)
	assert.Empty(t, db3.GetResourceID("jobs.my_job"))
	mustFinalize(t, &db3)
}
