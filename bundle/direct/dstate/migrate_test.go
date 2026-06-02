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

func TestMigrateStateLeavesDMSStateUntouched(t *testing.T) {
	// A DMS-upgraded state is already current; it must not be downgraded to the
	// baseline version, and the recorded DMS version must be preserved.
	db := &Database{Header: Header{StateVersion: dmsStateVersion, DmsVersion: dmsVersion}}
	require.NoError(t, migrateState(db))
	assert.Equal(t, dmsStateVersion, db.StateVersion)
	assert.Equal(t, dmsVersion, db.DmsVersion)
}

func TestMigrateStateRejectsNewerThanSupported(t *testing.T) {
	db := &Database{Header: Header{StateVersion: dmsStateVersion + 1}}
	err := migrateState(db)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "upgrade the CLI")
}

// TestStateSchemaVersions pins the state schema version constants. They are part
// of the on-disk format and a contract with older CLIs, so changing them must be
// deliberate. If you are intentionally bumping the baseline schema, this test is
// your checklist: follow the steps below, then update the assertions last.
//
// HOW TO BUMP THE BASELINE STATE VERSION (2 -> 4) AND RETIRE THE DMS SPECIAL-CASING
//
// Version 3 only exists because previewing DMS bumped the schema out-of-band via
// UpgradeToDMS, so non-preview bundles weren't forced to upgrade. The next baseline
// bump is when you delete all of that custom code and make 3 an ordinary version
// in the linear migration chain. Go to 4, not 3 (3 is already in the wild):
//
//  1. state.go: set currentStateVersion = 4. Delete the dmsStateVersion constant
//     and the UpgradeToDMS method. Version 3 is now reached only by migration,
//     like any other version. (Leave dmsVersion and Header.DmsVersion
//     alone; they are a separate concern, see step 4.)
//
//  2. migrate.go: change the upper-bound guard from "> dmsStateVersion" back to
//     "> currentStateVersion" (there is again only one current version). Add
//     stepwise migrations so every old state climbs to 4:
//     migrations[2] = v2 (non-DMS baseline)   -> v3
//     migrations[3] = v3 (former DMS preview) -> v4
//     A v2 state climbs 2->3->4 and a v3 state climbs 3->4, exactly like any
//     other version. Write a real transform for whatever the v4 change is.
//
//  3. deploy.go: delete the `if RecordDeploymentHistory { UpgradeToDMS() }` block.
//     The version is no longer bumped conditionally; every deploy writes the
//     baseline through the normal path.
//
//  4. dms_version is NOT part of this bump. It tracks the DMS *protocol* version
//     (dmsVersion), independent of the state schema version, and is enforced
//     by Open's WithDMS option (passed by cmd/bundle/utils/process.go only when the
//     bundle has opted into DMS). Leave the field, the constant, and that check in
//     place; if UpgradeToDMS was the only place stamping it, move that stamping into
//     the normal write path.
//
//  5. Tests: update the assertions below (the dmsStateVersion assertion and the
//     DMS-schema cases go away); add 2->3->4 and 3->4 migration tests.
//     TestMigrationsCoverBaseline fails until migrations[2] and migrations[3] exist.
//
// RELATED COVERAGE
//   - acceptance/bundle/state/permission_level_migration is a golden migration
//     fixture: it commits a real v1 state and asserts the migrated v2 output,
//     catching migration-correctness bugs (not just "did you mean to change this").
//   - acceptance/bundle/deploy/record-deployment-history/state-upgrade drives the
//     full lifecycle, including both rejections (older CLI vs newer state, and a
//     newer DMS version vs this CLI).
//   - bundle/invariant/continue_293 asserts the current CLI reads state written by
//     an older released CLI, so we never break reading older state.
func TestStateSchemaVersions(t *testing.T) {
	assert.Equal(t, 2, currentStateVersion)
	assert.Equal(t, 3, dmsStateVersion)
	assert.Equal(t, 1, dmsVersion)
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
