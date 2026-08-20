package dstate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/build"
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

// TestCLIVersionRecordsLastWriter pins that cli_version tracks the CLI that last
// wrote the state, not the one that created it. Previously the field was only set
// when the state was first created: the WAL header carried the deploying CLI's
// version but replay dropped it, so a state stayed pinned to its original writer
// no matter how many times a newer CLI deployed over it.
func TestCLIVersionRecordsLastWriter(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")

	// A state written by some older CLI.
	seed := `{"state_version":2,"cli_version":"0.1.2","lineage":"test-lineage","serial":1,"state":{}}`
	require.NoError(t, os.WriteFile(path, []byte(seed), 0o600))

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	require.NoError(t, db.SaveState("resources.jobs.my_job", "123", map[string]string{"k": "v"}, nil))
	mustFinalize(t, &db)

	var reopened DeploymentState
	require.NoError(t, reopened.Open(t.Context(), path, WithRecovery(false), WithWrite(false)))
	assert.Equal(t, build.GetInfo().Version, reopened.Data.CLIVersion)
	assert.Equal(t, 2, reopened.Data.Serial)
	mustFinalize(t, &reopened)
}

// TestHeaderOnlyWALDoesNotUpdateCLIVersion is the counterpart to the serial
// invariant below: a deploy that commits nothing does not persist a state file,
// so it must not claim to have written one.
func TestHeaderOnlyWALDoesNotUpdateCLIVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	walPath := path + walSuffix

	seed := `{"state_version":2,"cli_version":"0.1.2","lineage":"test-lineage","serial":1,"state":{}}`
	require.NoError(t, os.WriteFile(path, []byte(seed), 0o600))

	header := Header{Lineage: "test-lineage", Serial: 2, StateVersion: currentStateVersion, CLIVersion: build.GetInfo().Version}
	headerLine, err := json.Marshal(header)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(walPath, append(headerLine, '\n'), 0o600))

	var recovered DeploymentState
	require.NoError(t, recovered.Open(t.Context(), path, WithRecovery(true), WithWrite(false)))
	assert.Equal(t, "0.1.2", recovered.Data.CLIVersion, "a header-only WAL wrote no state, so the version must not move")
	assert.Equal(t, 1, recovered.Data.Serial)
	mustFinalize(t, &recovered)
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

// statFile stats through an open handle rather than by path: on Windows a
// path-based os.Stat resolves the file identity lazily, inside os.SameFile,
// which would re-resolve the path after the save and defeat the comparison in
// TestSaveReplacesStateFile.
func statFile(t *testing.T, path string) os.FileInfo {
	t.Helper()
	f, err := os.Open(path)
	require.NoError(t, err)
	defer f.Close()
	info, err := f.Stat()
	require.NoError(t, err)
	return info
}

// TestSaveReplacesStateFile pins that persisting state writes a new file and
// renames it over the previous one instead of truncating it in place. An
// in-place write that is interrupted leaves a state file that Open cannot
// parse, and Open fails on it before it looks at the WAL, so the intact WAL
// sitting next to it is never replayed and the deployment state is lost.
func TestSaveReplacesStateFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	var db DeploymentState
	require.NoError(t, db.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	require.NoError(t, db.SaveState("jobs.my_job", "123", map[string]string{"key": "val"}, nil))
	mustFinalize(t, &db)

	before := statFile(t, path)

	var db2 DeploymentState
	require.NoError(t, db2.Open(t.Context(), path, WithRecovery(true), WithWrite(true)))
	require.NoError(t, db2.SaveState("jobs.my_job", "456", map[string]string{"key": "val2"}, nil))
	mustFinalize(t, &db2)

	assert.False(t, os.SameFile(before, statFile(t, path)), "state file was written in place")

	// The rename leaves nothing behind: no temp file, and no WAL.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	assert.Equal(t, []string{"state.json"}, names)

	var db3 DeploymentState
	require.NoError(t, db3.Open(t.Context(), path, WithRecovery(false), WithWrite(false)))
	assert.Equal(t, "456", db3.GetResourceID("jobs.my_job"))
	assert.Equal(t, 2, db3.Data.Serial)
	mustFinalize(t, &db3)
}
