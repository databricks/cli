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
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))

	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{"key": "val"}, nil))
	mustFinalize(t, &db)

	// Re-open and verify persisted data.
	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(false), WithWrite(false)))
	assert.Equal(t, 1, db2.Data.Serial)
	assert.Equal(t, "123", db2.GetResourceID("jobs.my_job"))
	mustFinalize(t, &db2)
}

func TestRecordFeaturePersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	// RecordFeature must run before the WAL is started (UpgradeToWrite), so the
	// feature is captured in the WAL header and persisted.
	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(false)))
	db.RecordFeature(FeatureRecordDeploymentHistory)
	require.NoError(t, db.UpgradeToWrite())
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{"key": "val"}, nil))
	mustFinalize(t, &db)

	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(false), WithWrite(false)))
	assert.Equal(t, currentStateVersion, db2.Data.StateVersion)
	assert.Equal(t, featureMinCLIVersion[FeatureRecordDeploymentHistory], db2.Data.Features[FeatureRecordDeploymentHistory])
	mustFinalize(t, &db2)
}

func TestRecordFeaturePanicsAfterWALStarted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	assert.Panics(t, func() { db.RecordFeature(FeatureRecordDeploymentHistory) })
	mustFinalize(t, &db)
}

func TestOpenRejectsUnknownFeature(t *testing.T) {
	// A state recording a feature this CLI does not know is rejected, naming the
	// feature and the minimum CLI version the state recorded.
	path := filepath.Join(t.TempDir(), "state.json")
	data, err := json.Marshal(Database{Header: Header{
		StateVersion: currentStateVersion,
		Features:     map[string]string{"future_feature": "9.9.9"},
	}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))

	var db DeploymentState
	err = db.Open(t.Context(), path, WithRecovery(true), WithWrite(false))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `feature "future_feature"`)
	assert.Contains(t, err.Error(), "upgrade to 9.9.9 or newer")

	// A known feature loads fine.
	path2 := filepath.Join(t.TempDir(), "state.json")
	data, err = json.Marshal(Database{Header: Header{
		StateVersion: currentStateVersion,
		Features:     map[string]string{FeatureRecordDeploymentHistory: "0.0.0-dev"},
	}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path2, data, 0o600))

	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path2, WithRecovery(true), WithWrite(false)))
	mustFinalize(t, &db2)
}

func TestFinalizeWithNoEntriesDoesNotWriteStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	mustFinalize(t, &db)

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
	mustFinalize(t, &db)
}

func TestDeleteState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{}, nil))
	mustFinalize(t, &db)

	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	require.NoError(t, db2.DeleteState("jobs.my_job"))
	mustFinalize(t, &db2)

	var db3 DeploymentState
	require.NoError(t, db3.Open(t.Context(), path, WithRecovery(false), WithWrite(false)))
	assert.Equal(t, 2, db3.Data.Serial)
	assert.Empty(t, db3.GetResourceID("jobs.my_job"))
	mustFinalize(t, &db3)
}
