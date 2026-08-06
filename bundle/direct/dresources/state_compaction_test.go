package dresources

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// largeDashboard is a serialized_dashboard whose JSON encoding exceeds
	// stateHashPlaceholderLen, so it is actually compacted. Shared by the tests that assert
	// hashing happens, since a shorter value is deliberately left raw.
	largeDashboard = `{"pages":[{"name":"p1","displayName":"Page One","layout":[{"widget":{"name":"w1"}}]}]}`

	// smallDashboard is a serialized_dashboard whose JSON encoding fits within
	// stateHashPlaceholderLen, so hashing it would grow the state and it is persisted raw.
	smallDashboard = `{"pages":[{"name":"p1"}]}`
)

// requireLargeEnoughToHash asserts a fixture is on the hashed side of the size threshold.
// Whether a value is hashed depends on the length of its JSON encoding, so shrinking a
// fixture would silently leave it raw and make the tests below assert the opposite of what
// they were written for. Fail with an explicit message instead.
func requireLargeEnoughToHash(t *testing.T, content string) {
	t.Helper()
	encoded, err := json.Marshal(content)
	require.NoError(t, err)
	require.Greater(t, len(encoded), stateHashPlaceholderLen,
		"fixture must encode to more than %d bytes to be hashed; enlarge it", stateHashPlaceholderLen)
}

// requireTooSmallToHash is the inverse of requireLargeEnoughToHash: it asserts a fixture
// stays on the raw side of the threshold, so enlarging it cannot silently turn a
// persisted-raw test into a hashing test that passes for the wrong reason.
func requireTooSmallToHash(t *testing.T, content string) {
	t.Helper()
	encoded, err := json.Marshal(content)
	require.NoError(t, err)
	require.LessOrEqual(t, len(encoded), stateHashPlaceholderLen,
		"fixture must encode to at most %d bytes to be persisted raw; shrink it", stateHashPlaceholderLen)
}

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
	requireLargeEnoughToHash(t, largeDashboard)

	legacy := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: largeDashboard}}
	config := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: largeDashboard}}

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
	requireLargeEnoughToHash(t, largeDashboard)

	stringHash, err := hashStateValue(largeDashboard)
	require.NoError(t, err)
	require.IsType(t, "", stringHash)
	assert.True(t, strings.HasPrefix(stringHash.(string), stateHashPrefix))

	again, err := hashStateValue(largeDashboard)
	require.NoError(t, err)
	assert.Equal(t, stringHash, again)

	other, err := hashStateValue(strings.Replace(largeDashboard, "p1", "p2", 1))
	require.NoError(t, err)
	assert.NotEqual(t, stringHash, other)
}

// TestHashStateValueIdempotent verifies re-hashing an existing placeholder returns it
// unchanged, so re-compacting an already-compact state does not double-hash.
func TestHashStateValueIdempotent(t *testing.T) {
	requireLargeEnoughToHash(t, largeDashboard)

	hashed, err := hashStateValue(largeDashboard)
	require.NoError(t, err)

	again, err := hashStateValue(hashed)
	require.NoError(t, err)
	assert.Equal(t, hashed, again)
}

// TestHashStateValueSkipsSmallValues verifies a value whose JSON encoding would not
// shrink is left raw, so state never grows to hold a placeholder longer than the content
// it replaces. The boundary cases pin the comparison to the JSON encoding (which adds the
// surrounding quotes for a string) rather than the raw string length.
func TestHashStateValueSkipsSmallValues(t *testing.T) {
	// A string encodes with two surrounding quotes, so this is exactly at the limit.
	atLimit := strings.Repeat("x", stateHashPlaceholderLen-2)
	out, err := hashStateValue(atLimit)
	require.NoError(t, err)
	assert.Equal(t, atLimit, out)

	overLimit := atLimit + "x"
	out, err = hashStateValue(overLimit)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out.(string), stateHashPrefix))
	assert.Len(t, out, stateHashPlaceholderLen)
}

// TestCompactStateSkipsSmallField verifies the size check reaches CompactState, so a
// small field stays raw in the state it saves and on every side of the diff it compacts.
// Persisting it raw is what keeps the state smaller than a hash placeholder would.
func TestCompactStateSkipsSmallField(t *testing.T) {
	requireTooSmallToHash(t, smallDashboard)

	state := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: smallDashboard}}

	cfg := GetResourceConfig("dashboards")
	out, err := CompactState(cfg, state)
	require.NoError(t, err)
	assert.Equal(t, smallDashboard, out.(*DashboardState).SerializedDashboard)

	// The persisted form holds the content itself, and holding it costs less than the
	// placeholder that replacing it would have written.
	persisted, err := json.Marshal(out.(*DashboardState).SerializedDashboard)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(persisted), stateHashPlaceholderLen)

	// Compacting the already-raw state again leaves it alone, so repeated saves and every
	// side of the diff keep comparing content against content.
	out2, err := CompactState(cfg, out)
	require.NoError(t, err)
	assert.Equal(t, smallDashboard, out2.(*DashboardState).SerializedDashboard)
}

// TestCompactStateHashesLargeField is the counterpart of TestCompactStateSkipsSmallField:
// once the content outgrows a placeholder, compaction replaces it and the state shrinks.
func TestCompactStateHashesLargeField(t *testing.T) {
	requireLargeEnoughToHash(t, largeDashboard)

	state := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: largeDashboard}}

	out, err := CompactState(GetResourceConfig("dashboards"), state)
	require.NoError(t, err)
	compacted := out.(*DashboardState).SerializedDashboard
	require.IsType(t, "", compacted)
	assert.True(t, strings.HasPrefix(compacted.(string), stateHashPrefix))
	assert.Len(t, compacted, stateHashPlaceholderLen)

	// The whole point of hashing: the persisted form is smaller than the raw content.
	persisted, err := json.Marshal(compacted)
	require.NoError(t, err)
	raw, err := json.Marshal(largeDashboard)
	require.NoError(t, err)
	assert.Less(t, len(persisted), len(raw))
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
