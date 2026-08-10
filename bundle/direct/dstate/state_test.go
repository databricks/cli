package dstate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustFinalize(t *testing.T, db *DeploymentState) {
	t.Helper()
	_, err := db.Finalize(t.Context())
	require.NoError(t, err)
}

// fakeSink captures what the state writes report to DMS.
type fakeSink struct {
	ops []string
}

func (f *fakeSink) RecordOperation(ctx context.Context, resourceKey string, info OperationInfo, resourceID string, state json.RawMessage) {
	entry := fmt.Sprintf("%s %s id=%s state=%s", info.Action, resourceKey, resourceID, string(state))
	if info.InProgress {
		entry += " in_progress"
	}
	f.ops = append(f.ops, entry)
}

func TestStateWritesRecordOperations(t *testing.T) {
	tests := []struct {
		name  string
		write func(t *testing.T, db *DeploymentState)
		want  []string
	}{
		{
			// The service keeps one operation per resource per version, so the drop
			// opens it (no state: the old resource is gone and the new one does not
			// exist yet) and the save completes it. A deploy that stops in between
			// leaves the resource described as mid-recreate.
			name: "recreate reports both of its writes",
			write: func(t *testing.T, db *DeploymentState) {
				require.NoError(t, db.SaveState(t.Context(), "jobs.my_job", "123", map[string]string{"key": "old"}, nil, OperationInfo{Action: deployplan.Create}))
				require.NoError(t, db.DeleteState(t.Context(), "jobs.my_job", OperationInfo{Action: deployplan.Recreate, InProgress: true}))
				require.NoError(t, db.SaveState(t.Context(), "jobs.my_job", "456", map[string]string{"key": "new"}, nil, OperationInfo{Action: deployplan.Recreate}))
			},
			want: []string{
				`create jobs.my_job id=123 state={"state":{"key":"old"}}`,
				`recreate jobs.my_job id=123 state= in_progress`,
				`recreate jobs.my_job id=456 state={"state":{"key":"new"}}`,
			},
		},
		{
			name: "real delete reports the id it had and no state",
			write: func(t *testing.T, db *DeploymentState) {
				require.NoError(t, db.SaveState(t.Context(), "jobs.my_job", "123", map[string]string{}, nil, OperationInfo{Action: deployplan.Create}))
				require.NoError(t, db.DeleteState(t.Context(), "jobs.my_job", OperationInfo{Action: deployplan.Delete}))
			},
			want: []string{
				`create jobs.my_job id=123 state={"state":{}}`,
				`delete jobs.my_job id=123 state=`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "state.json")
			sink := &fakeSink{}

			var db DeploymentState
			require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))
			db.SetOperationSink(sink)

			tt.write(t, &db)
			mustFinalize(t, &db)

			assert.Equal(t, tt.want, sink.ops)
		})
	}
}

func TestStateWritesRecordNothingWithoutSink(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	// No sink: recording is off, and the writes still succeed.
	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))
	require.NoError(t, db.SaveState(t.Context(), "jobs.my_job", "123", map[string]string{}, nil, OperationInfo{Action: deployplan.Create}))
	require.NoError(t, db.DeleteState(t.Context(), "jobs.my_job", OperationInfo{Action: deployplan.Delete}))
	mustFinalize(t, &db)
}

func TestOpenSaveFinalizeRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))

	require.NoError(t, db.SaveState(t.Context(), "jobs.my_job", "123", map[string]string{"key": "val"}, nil, OperationInfo{Action: deployplan.Create}))
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
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))

	assert.Panics(t, func() {
		_ = db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil)
	})
	mustFinalize(t, &db)
}

func TestHeaderOnlyWALRecoveryDoesNotAdvanceSerial(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	walPath := path + walSuffix

	// Commit serial 1 with one resource.
	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))
	require.NoError(t, db.SaveState(t.Context(), "jobs.my_job", "123", map[string]string{}, nil, OperationInfo{Action: deployplan.Create}))
	mustFinalize(t, &db)

	var committed DeploymentState
	require.NoError(t, committed.Open(t.Context(), path, WithRecovery(false), WithWrite(false), nil))
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
	require.NoError(t, recovered.Open(t.Context(), path, WithRecovery(true), WithWrite(false), nil))
	assert.Equal(t, 1, recovered.Data.Serial)
	assert.Equal(t, "123", recovered.GetResourceID("jobs.my_job"))
	assert.NoFileExists(t, walPath)
	mustFinalize(t, &recovered)
}

// TestEmptyFeatureStateAcceptedWithoutFlippingVersion pins the special case that a
// featureStateVersion state with no features is accepted as-is — the on-disk version
// is left at featureStateVersion, not flipped down to currentStateVersion — and that
// a featureStateVersion state recording any feature is refused. This is scaffolding
// for the deferred version bump, special-cased to featureStateVersion only (see the
// featureStateVersion doc comment).
//
// When the baseline is actually bumped to featureStateVersion, this special case must
// go away. This test is the forcing function: it fails once featureStateVersion is
// removed, making the author decide what the post-bump behavior should be.
func TestEmptyFeatureStateAcceptedWithoutFlippingVersion(t *testing.T) {
	// The special case applies to featureStateVersion (3) only.
	require.Equal(t, 2, currentStateVersion, "when currentStateVersion is bumped, remove featureStateVersion and this special case")
	require.Equal(t, 3, featureStateVersion)

	empty := &Database{Header: Header{StateVersion: featureStateVersion}}
	require.NoError(t, migrateState(empty))
	assert.Equal(t, featureStateVersion, empty.StateVersion, "v3 + no features keeps its on-disk version, not flipped to v2")

	// v3 that records a feature is refused: this CLI does not understand features.
	withFeature := &Database{Header: Header{
		StateVersion: featureStateVersion,
		Features:     map[string]struct{}{"future_feature": {}},
	}}
	err := migrateState(withFeature)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "requires features this CLI does not support")
	assert.Contains(t, err.Error(), "future_feature")
	assert.Contains(t, err.Error(), "upgrade to the latest CLI version")
	assert.Contains(t, err.Error(), featuresDocURL)
}

func TestDeleteState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))
	require.NoError(t, db.SaveState(t.Context(), "jobs.my_job", "123", map[string]string{}, nil, OperationInfo{Action: deployplan.Create}))
	mustFinalize(t, &db)

	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(true), WithWrite(true), nil))
	require.NoError(t, db2.DeleteState(t.Context(), "jobs.my_job", OperationInfo{Action: deployplan.Delete}))
	mustFinalize(t, &db2)

	var db3 DeploymentState
	require.NoError(t, db3.Open(t.Context(), path, WithRecovery(false), WithWrite(false), nil))
	assert.Equal(t, 2, db3.Data.Serial)
	assert.Empty(t, db3.GetResourceID("jobs.my_job"))
	mustFinalize(t, &db3)
}

func TestGetOrInitLineageReadableBeforeWriteAndPersisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	// Fresh state opened read-only, as the deploy does before planning: no
	// lineage yet.
	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(false), nil))
	require.Empty(t, db.Data.Lineage)

	// GetOrInitLineage initializes the lineage and makes it readable before any
	// write (i.e. before planning), and is stable across calls.
	lineage := db.GetOrInitLineage()
	require.NotEmpty(t, lineage)
	require.Equal(t, lineage, db.GetOrInitLineage())

	// Upgrading to write reuses the same lineage (it goes into the WAL header),
	// and a write makes it durable.
	require.NoError(t, db.UpgradeToWrite())
	require.NoError(t, db.SaveState(t.Context(), "jobs.my_job", "123", map[string]string{}, nil, OperationInfo{Action: deployplan.Create}))
	mustFinalize(t, &db)

	// Re-open: the persisted lineage matches the one read before the write.
	var reopened DeploymentState
	require.NoError(t, reopened.Open(t.Context(), path, WithRecovery(false), WithWrite(false), nil))
	assert.Equal(t, lineage, reopened.Data.Lineage)
	mustFinalize(t, &reopened)
}
