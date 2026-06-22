package dresources

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHashedInStateImpliesIgnoreRemoteChanges enforces the core invariant of the
// hashed_in_state mechanism: a field stored as a content hash can only ever detect a
// LOCAL change (saved-hash vs config-hash). Its hash never equals the full-content
// remote value, so its remote diff is meaningless and must be discarded — i.e. the
// field must also be ignore_remote_changes, otherwise every plan would report a
// permanent spurious update for it. This holds for every resource, not just dashboards.
func TestHashedInStateImpliesIgnoreRemoteChanges(t *testing.T) {
	for _, cfg := range []*Config{MustLoadConfig(), MustLoadGeneratedConfig()} {
		for resourceType, rc := range cfg.Resources {
			for _, hashed := range rc.HashedInState {
				path, err := structpath.ParsePath(hashed.Field.String())
				require.NoError(t, err, "%s: invalid hashed_in_state field %q", resourceType, hashed.Field)

				covered := false
				for _, ignore := range rc.IgnoreRemoteChanges {
					if path.HasPatternPrefix(ignore.Field) {
						covered = true
						break
					}
				}
				assert.True(t, covered,
					"%s: field %q is hashed_in_state but not ignore_remote_changes (a hashed field only detects local changes; its remote diff must be ignored)",
					resourceType, hashed.Field)
			}
		}
	}
}

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
