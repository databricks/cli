package dstate

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateStateLeavesCurrentUntouched(t *testing.T) {
	db := &Database{Header: Header{StateVersion: currentStateVersion}}
	require.NoError(t, migrateState(db))
	assert.Equal(t, currentStateVersion, db.StateVersion)
}

func TestMigrateStateUpgradesLegacyToCurrent(t *testing.T) {
	// A legacy state migrates forward to the current version (the v2->v3 step adds
	// the feature list, which is absent by default).
	db := &Database{Header: Header{StateVersion: 2}}
	require.NoError(t, migrateState(db))
	assert.Equal(t, currentStateVersion, db.StateVersion)
}

func TestMigrateStateRejectsNewerThanSupported(t *testing.T) {
	db := &Database{Header: Header{StateVersion: currentStateVersion + 1}}
	err := migrateState(db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upgrade the CLI")
}

// TestStateSchemaVersion pins currentStateVersion. It is part of the on-disk
// format and a contract with older CLIs, so changing it must be deliberate.
//
// Bump it only for a structural schema change that older CLIs cannot read: add a
// migration to the migrations map (TestMigrationsCoverBaseline enforces full
// coverage) and update the assertion below. For an additive capability that does
// not change the structure, record a feature flag instead (see knownFeatures and
// RecordFeature) — that lets older CLIs fail with an actionable "upgrade to X"
// message without a version bump.
//
// RELATED COVERAGE
//   - acceptance/bundle/state/permission_level_migration: golden v1->v2 migration.
//   - acceptance/bundle/state/unknown_feature: a state requiring an unknown feature
//     is rejected with the recorded minimum CLI version.
//   - bundle/invariant/continue_293: the current CLI reads state written by an
//     older released CLI.
func TestStateSchemaVersion(t *testing.T) {
	assert.Equal(t, 3, currentStateVersion)
}

// TestMigrationsCoverBaseline guards a baseline bump: every state version below
// currentStateVersion must have a migration to the next version, so migrateState
// can always climb a legacy state up to the baseline. A bump that forgets a
// migration fails here instead of at a user's deploy.
func TestMigrationsCoverBaseline(t *testing.T) {
	for v := range currentStateVersion {
		assert.Containsf(t, migrations, v, "missing migration for state version %d", v)
	}
}

func TestCheckSupportedFeatures(t *testing.T) {
	// Known features (and none at all) are accepted; an unknown feature is rejected
	// with its name and the recorded minimum CLI version.
	require.NoError(t, checkSupportedFeatures(&Database{}))
	require.NoError(t, checkSupportedFeatures(&Database{Header: Header{
		Features: map[string]string{FeatureRecordDeploymentHistory: "0.0.0-dev"},
	}}))

	err := checkSupportedFeatures(&Database{Header: Header{
		Features: map[string]string{"future_feature": "9.9.9"},
	}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), `feature "future_feature"`)
	assert.Contains(t, err.Error(), "upgrade to 9.9.9 or newer")
}
