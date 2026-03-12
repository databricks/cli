package dstate

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateV1ToV2_PermissionsEntry(t *testing.T) {
	db := Database{
		StateVersion: 1,
		CLIVersion:   "0.0.0-dev",
		Lineage:      "test",
		Serial:       1,
		State: map[string]ResourceEntry{
			"resources.jobs.my_job": {
				ID:    "123",
				State: json.RawMessage(`{"name": "my job"}`),
			},
			"resources.jobs.my_job.permissions": {
				ID: "/jobs/123",
				State: json.RawMessage(`{
					"object_id": "/jobs/123",
					"permissions": [
						{"permission_level": "CAN_VIEW", "group_name": "viewers"},
						{"permission_level": "IS_OWNER", "user_name": "tester@databricks.com"}
					]
				}`),
			},
		},
	}

	err := migrateState(&db)
	require.NoError(t, err)
	assert.Equal(t, 2, db.StateVersion)

	// Non-permissions entry should be unchanged.
	assert.Equal(t, `{"name": "my job"}`, string(db.State["resources.jobs.my_job"].State))

	// Permissions entry should be migrated.
	var result newPermissionsStateV2
	err = json.Unmarshal(db.State["resources.jobs.my_job.permissions"].State, &result)
	require.NoError(t, err)
	assert.Equal(t, "/jobs/123", result.ObjectID)
	require.Len(t, result.EmbeddedSlice, 2)
	assert.Equal(t, "CAN_VIEW", result.EmbeddedSlice[0].Level)
	assert.Equal(t, "viewers", result.EmbeddedSlice[0].GroupName)
	assert.Equal(t, "IS_OWNER", result.EmbeddedSlice[1].Level)
	assert.Equal(t, "tester@databricks.com", result.EmbeddedSlice[1].UserName)
}

func TestMigrateV1ToV2_AlreadyNewFormat(t *testing.T) {
	// State that already uses new format (e.g., was created by new CLI but version wasn't bumped).
	db := Database{
		StateVersion: 1,
		CLIVersion:   "0.0.0-dev",
		Lineage:      "test",
		Serial:       1,
		State: map[string]ResourceEntry{
			"resources.jobs.my_job.permissions": {
				ID: "/jobs/123",
				State: json.RawMessage(`{
					"object_id": "/jobs/123",
					"_": [
						{"level": "CAN_VIEW", "group_name": "viewers"}
					]
				}`),
			},
		},
	}

	err := migrateState(&db)
	require.NoError(t, err)
	assert.Equal(t, 2, db.StateVersion)

	// Should pass through unchanged (old.Permissions is empty, so raw is returned as-is).
	var result newPermissionsStateV2
	err = json.Unmarshal(db.State["resources.jobs.my_job.permissions"].State, &result)
	require.NoError(t, err)
	assert.Equal(t, "CAN_VIEW", result.EmbeddedSlice[0].Level)
}

func TestMigrateState_CurrentVersion(t *testing.T) {
	db := Database{
		StateVersion: currentStateVersion,
		State:        map[string]ResourceEntry{},
	}

	err := migrateState(&db)
	require.NoError(t, err)
	assert.Equal(t, currentStateVersion, db.StateVersion)
}

func TestMigrateState_Version0(t *testing.T) {
	// Version 0 means state_version was absent; should be treated like version 1.
	db := Database{
		StateVersion: 0,
		State:        map[string]ResourceEntry{},
	}

	err := migrateState(&db)
	require.NoError(t, err)
	assert.Equal(t, currentStateVersion, db.StateVersion)
}
