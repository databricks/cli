package dresources

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHashedInStateFieldsAreTopLevel guards the shallow-copy assumption in CompactState:
// every hashed_in_state path declared in resources.yml must be a top-level field. A nested
// path would be mutated through memory shared with the deploy value (see CompactState), so
// this fails CI the moment such a declaration is added instead of corrupting state at runtime.
func TestHashedInStateFieldsAreTopLevel(t *testing.T) {
	for name, rc := range MustLoadConfig().Resources {
		for _, field := range rc.HashedInState {
			path, err := structpath.ParsePath(field)
			require.NoError(t, err, "%s: hashed_in_state field %q", name, field)
			assert.Equal(t, 1, path.Len(), "%s: hashed_in_state field %q must be a top-level field", name, field)
		}
	}
}

// TestCompactStateRejectsNestedField verifies CompactState errors on a nested
// hashed_in_state path rather than mutating memory shared with the deploy value.
func TestCompactStateRejectsNestedField(t *testing.T) {
	cfg := &ResourceLifecycleConfig{HashedInState: []string{"foo.bar"}}
	state := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: `{"a":1}`}}

	_, err := CompactState(cfg, state)
	require.ErrorContains(t, err, "must be a top-level field")
}

// TestCompactStateNoDeclaredFields verifies CompactState is a no-op for a resource
// type with no hashed_in_state declaration and for a nil config, returning the same
// value untouched.
func TestCompactStateNoDeclaredFields(t *testing.T) {
	state := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: `{"a":1}`}}

	out, err := CompactState(GetResourceConfig("jobs"), state)
	require.NoError(t, err)
	assert.Same(t, state, out.(*DashboardState))

	out, err = CompactState(nil, state)
	require.NoError(t, err)
	assert.Same(t, state, out.(*DashboardState))
}

// TestCompactStateMigratesLegacyFullContent verifies that a legacy state holding the
// full serialized_dashboard and a config holding identical content compact to the
// same hash, so a diff computed after hashing-on-read shows no spurious change and
// the next save rewrites the state compactly.
func TestCompactStateMigratesLegacyFullContent(t *testing.T) {
	content := `{"pages":[{"name":"p1"}]}`
	legacy := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: content}}
	config := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: content}}

	cfg := GetResourceConfig("dashboards")
	compactedLegacy, err := CompactState(cfg, legacy)
	require.NoError(t, err)
	compactedConfig, err := CompactState(cfg, config)
	require.NoError(t, err)

	legacyHash := compactedLegacy.(*DashboardState).SerializedDashboard
	assert.Equal(t, compactedConfig.(*DashboardState).SerializedDashboard, legacyHash)
	assert.True(t, strings.HasPrefix(legacyHash.(string), stateHashPrefix))
}

// TestHashStateValue verifies hashStateValue adds the state hash prefix and produces
// a stable placeholder: the same content always hashes to the same value and
// different content differs.
func TestHashStateValue(t *testing.T) {
	stringHash, err := hashStateValue("hello")
	require.NoError(t, err)
	require.IsType(t, "", stringHash)
	assert.True(t, strings.HasPrefix(stringHash.(string), stateHashPrefix))

	again, err := hashStateValue("hello")
	require.NoError(t, err)
	assert.Equal(t, stringHash, again)

	other, err := hashStateValue("world")
	require.NoError(t, err)
	assert.NotEqual(t, stringHash, other)
}

// TestHashStateValueIdempotent verifies re-hashing an existing placeholder returns it
// unchanged, so re-compacting an already-compact state does not double-hash.
func TestHashStateValueIdempotent(t *testing.T) {
	hashed, err := hashStateValue("some content")
	require.NoError(t, err)

	again, err := hashStateValue(hashed)
	require.NoError(t, err)
	assert.Equal(t, hashed, again)
}

// TestHashStateValueEmptyAndNil verifies empty and nil values pass through unchanged,
// since there is nothing to hash.
func TestHashStateValueEmptyAndNil(t *testing.T) {
	empty, err := hashStateValue("")
	require.NoError(t, err)
	assert.Empty(t, empty)

	null, err := hashStateValue(nil)
	require.NoError(t, err)
	assert.Nil(t, null)
}
