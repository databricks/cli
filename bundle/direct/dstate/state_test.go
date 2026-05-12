package dstate

import (
	"path/filepath"
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

func TestPanicOnDoubleOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(path))

	assert.Panics(t, func() {
		_ = db.Open(path)
	})
}

func TestDeleteState(t *testing.T) {
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
