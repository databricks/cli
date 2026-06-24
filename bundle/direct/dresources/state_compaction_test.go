package dresources

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompactStateNoDeclaredFields(t *testing.T) {
	state := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: `{"a":1}`}}

	// A resource type with no hashed_in_state declaration returns the value untouched.
	out, err := CompactState(GetResourceConfig("jobs"), state)
	require.NoError(t, err)
	assert.Same(t, state, out.(*DashboardState))

	// A nil config is also a no-op.
	out, err = CompactState(nil, state)
	require.NoError(t, err)
	assert.Same(t, state, out.(*DashboardState))
}

func TestCompactStateMigratesLegacyFullContent(t *testing.T) {
	// A pre-existing state still holds the full serialized_dashboard; the matching config
	// holds identical content. Compaction must map both to the same hash, so a diff computed
	// after the legacy state is hashed-on-read shows no spurious change (and the next save
	// rewrites the state compactly).
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

func TestHashStateValue(t *testing.T) {
	stringHash, err := hashStateValue("hello")
	require.NoError(t, err)
	require.IsType(t, "", stringHash)
	assert.True(t, strings.HasPrefix(stringHash.(string), stateHashPrefix))

	// Same content always hashes to the same value.
	again, err := hashStateValue("hello")
	require.NoError(t, err)
	assert.Equal(t, stringHash, again)

	// Different content hashes differently.
	other, err := hashStateValue("world")
	require.NoError(t, err)
	assert.NotEqual(t, stringHash, other)

	// A map hashes deterministically regardless of literal key order.
	mapHash, err := hashStateValue(map[string]any{"a": 1, "b": 2})
	require.NoError(t, err)
	mapHash2, err := hashStateValue(map[string]any{"b": 2, "a": 1})
	require.NoError(t, err)
	assert.Equal(t, mapHash, mapHash2)
}

func TestHashStateValueIdempotent(t *testing.T) {
	hashed, err := hashStateValue("some content")
	require.NoError(t, err)

	// Re-hashing a placeholder returns it unchanged.
	again, err := hashStateValue(hashed)
	require.NoError(t, err)
	assert.Equal(t, hashed, again)
}

func TestHashStateValueEmptyAndNil(t *testing.T) {
	empty, err := hashStateValue("")
	require.NoError(t, err)
	assert.Empty(t, empty)

	null, err := hashStateValue(nil)
	require.NoError(t, err)
	assert.Nil(t, null)
}
