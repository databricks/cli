package dstate

import (
	"encoding/json"
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
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), WithDMS(false)))

	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{"key": "val"}, nil))
	mustFinalize(t, &db)

	// Re-open and verify persisted data.
	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(false), WithWrite(false), WithDMS(false)))
	assert.Equal(t, 1, db2.Data.Serial)
	assert.Equal(t, "123", db2.GetResourceID("jobs.my_job"))
	mustFinalize(t, &db2)
}

func TestUpgradeToDMSPersistsVersions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	// UpgradeToDMS must run before the WAL is started (UpgradeToWrite), so the
	// bumped version is captured in the WAL header.
	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(false), WithDMS(false)))
	db.UpgradeToDMS()
	require.NoError(t, db.UpgradeToWrite())
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{"key": "val"}, nil))
	mustFinalize(t, &db)

	// Re-open and verify the upgraded schema version and DMS version persisted,
	// and that loading the upgraded state does not error or downgrade it.
	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(false), WithWrite(false), WithDMS(false)))
	assert.Equal(t, dmsStateVersion, db2.Data.StateVersion)
	assert.Equal(t, dmsVersion, db2.Data.DmsVersion)
	mustFinalize(t, &db2)
}

func TestUpgradeToDMSPanicsAfterWALStarted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), WithDMS(false)))
	assert.Panics(t, db.UpgradeToDMS)
	mustFinalize(t, &db)
}

func TestOpenWithDMSRejectsNewerDmsVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	data, err := json.Marshal(Database{Header: Header{StateVersion: dmsStateVersion, DmsVersion: dmsVersion + 1}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))

	// WithDMS(true): a state with a newer DMS protocol version is rejected.
	var db DeploymentState
	err = db.Open(t.Context(), path, WithRecovery(true), WithWrite(false), WithDMS(true))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "record_deployment_history state version")
	assert.Contains(t, err.Error(), "upgrade the CLI")

	// WithDMS(false): the same state loads fine; the check is gated on opt-in.
	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(true), WithWrite(false), WithDMS(false)))
	mustFinalize(t, &db2)
}

func TestFinalizeWithNoEntriesDoesNotWriteStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), WithDMS(false)))
	mustFinalize(t, &db)

	_, err := os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestPanicOnDoubleOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), WithDMS(false)))

	assert.Panics(t, func() {
		_ = db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), WithDMS(false))
	})
	mustFinalize(t, &db)
}

func TestDeleteState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), WithDMS(false)))
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{}, nil))
	mustFinalize(t, &db)

	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(true), WithWrite(true), WithDMS(false)))
	require.NoError(t, db2.DeleteState("jobs.my_job"))
	mustFinalize(t, &db2)

	var db3 DeploymentState
	require.NoError(t, db3.Open(t.Context(), path, WithRecovery(false), WithWrite(false), WithDMS(false)))
	assert.Equal(t, 2, db3.Data.Serial)
	assert.Empty(t, db3.GetResourceID("jobs.my_job"))
	mustFinalize(t, &db3)
}
