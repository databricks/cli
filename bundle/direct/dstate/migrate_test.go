package dstate

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateV2ToV3GrantsEntry(t *testing.T) {
	input := json.RawMessage(`{
		"securable_type": "catalog",
		"full_name": "main",
		"grants": [
			{"principal": "user@example.com", "privileges": ["USE_CATALOG"]}
		]
	}`)

	result, err := migrateGrantsEntry(input)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, json.Unmarshal(result, &parsed))

	assert.Equal(t, "catalog", parsed["securable_type"])
	assert.Equal(t, "main", parsed["full_name"])
	assert.Nil(t, parsed["grants"], "old 'grants' key should be removed")
	assert.NotNil(t, parsed["__embed__"], "'__embed__' key should be present")
}

func TestMigrateV2ToV3AlreadyMigrated(t *testing.T) {
	input := json.RawMessage(`{
		"securable_type": "catalog",
		"full_name": "main",
		"__embed__": [
			{"principal": "user@example.com", "privileges": ["USE_CATALOG"]}
		]
	}`)

	result, err := migrateGrantsEntry(input)
	require.NoError(t, err)

	// Should pass through unchanged since there's no "grants" field.
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(result, &parsed))
	assert.NotNil(t, parsed["__embed__"])
}

func TestMigrateV2ToV3FullDatabase(t *testing.T) {
	db := &Database{
		StateVersion: 2,
		State: map[string]ResourceEntry{
			"resources.catalogs.my_cat.grants": {
				ID: "catalog/main",
				State: json.RawMessage(`{
					"securable_type": "catalog",
					"full_name": "main",
					"grants": [
						{"principal": "user@example.com", "privileges": ["USE_CATALOG"]}
					]
				}`),
			},
			"resources.jobs.my_job": {
				ID:    "123",
				State: json.RawMessage(`{"job_id": 123}`),
			},
		},
	}

	err := migrateState(db)
	require.NoError(t, err)
	assert.Equal(t, currentStateVersion, db.StateVersion)

	// Grants entry should be migrated.
	var grantsState map[string]any
	require.NoError(t, json.Unmarshal(db.State["resources.catalogs.my_cat.grants"].State, &grantsState))
	assert.NotNil(t, grantsState["__embed__"])
	assert.Nil(t, grantsState["grants"])

	// Non-grants entry should be unchanged.
	var jobState map[string]any
	require.NoError(t, json.Unmarshal(db.State["resources.jobs.my_job"].State, &jobState))
	assert.Equal(t, float64(123), jobState["job_id"])
}
