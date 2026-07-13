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

func TestFinalizeWithNoEntriesDoesNotWriteStateFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	mustFinalize(t, &db)

	_, err := os.Stat(path)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestExportStateFromDataJobRunJobID(t *testing.T) {
	data := Database{
		State: map[string]ResourceEntry{
			"resources.job_runs.my_run": {
				ID:    "456",
				State: json.RawMessage(`{"job_id": 123}`),
			},
			// A permissions sub-resource entry shares the ".job_runs." infix but
			// is not a run. Its state even carries a job_id to prove we match on
			// the exact resource type, not a substring of the key.
			"resources.job_runs.my_run.permissions": {
				ID:    "456",
				State: json.RawMessage(`{"job_id": 999}`),
			},
		},
	}

	result := ExportStateFromData(data)

	assert.Equal(t, int64(123), result["resources.job_runs.my_run"].JobID)
	assert.Equal(t, int64(0), result["resources.job_runs.my_run.permissions"].JobID)
}

func TestExportStateFromDataDashboardEtag(t *testing.T) {
	data := Database{
		State: map[string]ResourceEntry{
			"resources.dashboards.my_dash": {
				ID:    "abc",
				State: json.RawMessage(`{"etag": "v1"}`),
			},
			// A permissions sub-resource entry shares the ".dashboards." infix but
			// is not a dashboard; its etag must not be lifted onto it.
			"resources.dashboards.my_dash.permissions": {
				ID:    "abc",
				State: json.RawMessage(`{"etag": "v2"}`),
			},
		},
	}

	result := ExportStateFromData(data)

	assert.Equal(t, "v1", result["resources.dashboards.my_dash"].ETag)
	assert.Empty(t, result["resources.dashboards.my_dash.permissions"].ETag)
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

func TestHeaderOnlyWALRecoveryDoesNotAdvanceSerial(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	walPath := path + walSuffix

	// Commit serial 1 with one resource.
	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{}, nil))
	mustFinalize(t, &db)

	var committed DeploymentState
	require.NoError(t, committed.Open(t.Context(), path, WithRecovery(false), WithWrite(false)))
	lineage := committed.Data.Lineage
	require.Equal(t, 1, committed.Data.Serial)
	mustFinalize(t, &committed)

	// A deploy that opens the WAL for write but commits nothing (e.g. planning
	// fails after UpgradeToWrite) leaves a header-only WAL behind, here at the
	// expected serial 2. Recovering it must not advance the serial past the
	// committed 1, otherwise a second such failed deploy would write its header
	// at serial 3 and be rejected as ahead of the committed state.
	header := Header{Lineage: lineage, Serial: 2, StateVersion: currentStateVersion}
	headerLine, err := json.Marshal(header)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(walPath, append(headerLine, '\n'), 0o600))

	var recovered DeploymentState
	require.NoError(t, recovered.Open(t.Context(), path, WithRecovery(true), WithWrite(false)))
	assert.Equal(t, 1, recovered.Data.Serial)
	assert.Equal(t, "123", recovered.GetResourceID("jobs.my_job"))
	assert.NoFileExists(t, walPath)
	mustFinalize(t, &recovered)
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

func TestGetOrInitLineageReadableBeforeWriteAndPersisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	// Fresh state opened read-only, as the deploy does before planning: no
	// lineage yet.
	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(false)))
	require.Empty(t, db.Data.Lineage)

	// GetOrInitLineage initializes the lineage and makes it readable before any
	// write (i.e. before planning), and is stable across calls.
	lineage := db.GetOrInitLineage()
	require.NotEmpty(t, lineage)
	require.Equal(t, lineage, db.GetOrInitLineage())

	// Upgrading to write reuses the same lineage (it goes into the WAL header),
	// and a write makes it durable.
	require.NoError(t, db.UpgradeToWrite())
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{}, nil))
	mustFinalize(t, &db)

	// Re-open: the persisted lineage matches the one read before the write.
	var reopened DeploymentState
	require.NoError(t, reopened.Open(t.Context(), path, WithRecovery(false), WithWrite(false)))
	assert.Equal(t, lineage, reopened.Data.Lineage)
	mustFinalize(t, &reopened)
}
