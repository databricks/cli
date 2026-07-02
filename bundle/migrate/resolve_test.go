package migrate_test

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/bundle/migrate"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// noWarn is a warnf that drops messages; these cases never hit the warning path.
func noWarn(string, ...any) {}

// state with src job having int and bool fields set.
func testState() migrate.TFStateAttrs {
	return migrate.TFStateAttrs{
		"databricks_job": {
			"src": json.RawMessage(`{
				"id": "111",
				"max_concurrent_runs": 4,
				"always_running": true
			}`),
			"dst": json.RawMessage(`{
				"id": "222",
				"max_concurrent_runs": 4,
				"always_running": true
			}`),
		},
	}
}

// TestResolveFieldRefInt proves that when Method B (template evaluation) wins for
// an int field, the returned string value is still usable: structaccess.Set must
// parse it back to int and not error.
func TestResolveFieldRefInt(t *testing.T) {
	state := testState()
	// Remove dst from state so Method A fails and Method B must be used.
	delete(state["databricks_job"], "dst")

	fieldPath, err := structpath.ParsePath("max_concurrent_runs")
	require.NoError(t, err)

	value, err := migrate.ResolveFieldRef(state, "jobs", "dst", fieldPath,
		"${resources.jobs.src.max_concurrent_runs}", noWarn)
	require.NoError(t, err)

	// Method B succeeds: returns string "4". Verify Set converts it to int.
	type target struct {
		MaxConcurrentRuns int `json:"max_concurrent_runs"`
	}
	s := &target{}
	err = structaccess.Set(s, fieldPath, value)
	assert.NoError(t, err, "Set should parse string %q into int field", value)
	assert.Equal(t, 4, s.MaxConcurrentRuns)
}

// TestResolveFieldRefBool is the same for a bool field.
func TestResolveFieldRefBool(t *testing.T) {
	state := testState()
	delete(state["databricks_job"], "dst")

	fieldPath, err := structpath.ParsePath("always_running")
	require.NoError(t, err)

	value, err := migrate.ResolveFieldRef(state, "jobs", "dst", fieldPath,
		"${resources.jobs.src.always_running}", noWarn)
	require.NoError(t, err)

	type target struct {
		AlwaysRunning bool `json:"always_running"`
	}
	s := &target{}
	err = structaccess.Set(s, fieldPath, value)
	assert.NoError(t, err, "Set should parse string %q into bool field", value)
	assert.True(t, s.AlwaysRunning)
}
