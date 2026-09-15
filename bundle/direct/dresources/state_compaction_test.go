package dresources

import (
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structaccess"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// largeDashboard is a serialized_dashboard longer than stateHashPlaceholderLen, so it is
	// actually compacted.
	largeDashboard = `{"pages":[{"name":"p1","displayName":"Page One","layout":[{"widget":{"name":"w1"}}]}]}`

	// smallDashboard is a serialized_dashboard no longer than stateHashPlaceholderLen, so
	// hashing it would grow the state and it is persisted raw.
	smallDashboard = `{"pages":[{"name":"p1"}]}`
)

// requireLargeEnoughToHash asserts a fixture is on the hashed side of the size threshold.
// Whether a value is hashed depends on its length, so shrinking a fixture would silently
// leave it raw and make the tests below assert the opposite of what they were written for.
func requireLargeEnoughToHash(t *testing.T, content string) {
	t.Helper()
	require.Greater(t, len(content), stateHashPlaceholderLen,
		"fixture must be more than %d bytes to be hashed; enlarge it", stateHashPlaceholderLen)
}

// requireTooSmallToHash is the inverse of requireLargeEnoughToHash: it asserts a fixture
// stays on the raw side of the threshold, so enlarging it cannot silently turn a
// persisted-raw test into a hashing test that passes for the wrong reason.
func requireTooSmallToHash(t *testing.T, content string) {
	t.Helper()
	require.LessOrEqual(t, len(content), stateHashPlaceholderLen,
		"fixture must be at most %d bytes to be persisted raw; shrink it", stateHashPlaceholderLen)
}

// TestHashedFieldsAreValid checks every hashed_fields path in resources.yml is a top-level
// field (CompactState's shallow copy only isolates those) and a real field on the state type
// (a typo parses fine but resolves to nothing, so CompactState silently skips it).
func TestHashedFieldsAreValid(t *testing.T) {
	for name, rc := range MustLoadConfig().Resources {
		if len(rc.HashedFields) == 0 {
			continue
		}
		adapter, err := NewAdapter(SupportedResources[name], name, nil)
		require.NoError(t, err, "%s: failed to create adapter", name)
		for _, field := range rc.HashedFields {
			path, err := structpath.ParsePath(field)
			require.NoError(t, err, "%s: hashed_fields field %q", name, field)
			assert.Equal(t, 1, path.Len(), "%s: hashed_fields field %q must be a top-level field", name, field)
			assert.NoError(t, structaccess.ValidatePath(adapter.StateType(), path),
				"%s: hashed_fields field %q must be an actual field on the state type", name, field)
		}
	}
}

// TestCompactStateRejectsNestedField verifies CompactState errors on a nested
// hashed_fields path rather than mutating memory shared with the deploy value.
func TestCompactStateRejectsNestedField(t *testing.T) {
	cfg := &ResourceLifecycleConfig{HashedFields: []string{"foo.bar"}}
	state := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: `{"a":1}`}}

	_, err := CompactState(cfg, state)
	require.ErrorContains(t, err, "must be a top-level field")
}

// TestCompactStateRejectsNonStringField verifies CompactState panics when a hashed_fields
// value is not a string. hashed_fields values are serialized to a string before the deploy
// engine runs, so a non-string here is a broken invariant, not a user error.
func TestCompactStateRejectsNonStringField(t *testing.T) {
	state := &DashboardState{DashboardConfig: resources.DashboardConfig{
		SerializedDashboard: map[string]any{"pages": []any{}},
	}}

	assert.Panics(t, func() {
		_, _ = CompactState(GetResourceConfig("dashboards"), state)
	})
}

// TestCompactStateSkipsNilField verifies an unset hashed_fields value passes through
// untouched, since there is nothing to hash.
func TestCompactStateSkipsNilField(t *testing.T) {
	state := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: nil}}

	out, err := CompactState(GetResourceConfig("dashboards"), state)
	require.NoError(t, err)
	assert.Nil(t, out.(*DashboardState).SerializedDashboard)
}

// TestCompactStateNoDeclaredFields verifies CompactState is a no-op for a resource
// type with no hashed_fields declaration and for a nil config, returning the same
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

// TestStateHashing verifies, per fixture, that hashStateValue maps content to the exact
// expected value (a hardcoded sha256:<hex> placeholder when hashing pays off, the content
// itself otherwise), that isStateHashPlaceholder recognizes only the strings this package
// produces, and that CompactState of a dashboard's serialized_dashboard yields the same.
func TestStateHashing(t *testing.T) {
	requireTooSmallToHash(t, smallDashboard)

	// A raw string of exactly stateHashPlaceholderLen bytes sits at the limit; one more tips
	// it over. hex64 is a valid digest body; placeholder is a full, valid placeholder.
	atLimit := strings.Repeat("x", stateHashPlaceholderLen)
	hex64 := strings.Repeat("a", sha256.Size*2)
	placeholder := stateHashPrefix + hex64

	cases := []struct {
		name          string
		content       string
		want          string
		isPlaceholder bool
	}{
		{
			name: "multiline",
			content: `
first line of content
second line of content
third line of content
fourth line of content
`,
			want: "sha256:c74674070e73f13131f1611d89ecf2ff74c7adf5f998da1321b026a7f154bf1e",
		},
		{
			name:    "over_size_limit",
			content: atLimit + "x",
			want:    "sha256:5bd2fd7019be4014a135cbc4fc4aecc894861782ec797467a7cd0909e98f76f2",
		},
		{
			// Shares the prefix but is one byte too long to be a placeholder, so it hashes.
			name:    "too_long",
			content: stateHashPrefix + strings.Repeat("a", 65),
			want:    "sha256:ee9921c0587e224847f0833ce1ef61f0d92295bc283a268caff458cf98e87cf3",
		},
		{
			name:          "valid_placeholder",
			content:       placeholder,
			want:          placeholder,
			isPlaceholder: true,
		},
		{
			name:    "empty",
			content: "",
			want:    "",
		},
		{
			name:    "small_stays_raw",
			content: smallDashboard,
			want:    smallDashboard,
		},
		{
			name:    "at_size_limit",
			content: atLimit,
			want:    atLimit,
		},
		{
			name:    "too_short",
			content: stateHashPrefix + strings.Repeat("a", 63),
			want:    stateHashPrefix + strings.Repeat("a", 63),
		},
		{
			name:    "uppercase",
			content: stateHashPrefix + strings.Repeat("A", 64),
			want:    stateHashPrefix + strings.Repeat("A", 64),
		},
		{
			name:    "no_prefix",
			content: hex64,
			want:    hex64,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hashStateValue(tc.content)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.isPlaceholder, isStateHashPlaceholder(tc.content))
			// Idempotent: re-hashing the output is a no-op, so re-compacting never double-hashes.
			assert.Equal(t, tc.want, hashStateValue(got))
			if got != tc.content {
				// Hashing only ever happens when it shrinks the value.
				assert.Less(t, len(got), len(tc.content))
			}

			// CompactState delegates to hashStateValue, so a dashboard's serialized_dashboard
			// compacts to the same value.
			state := &DashboardState{DashboardConfig: resources.DashboardConfig{SerializedDashboard: tc.content}}
			out, err := CompactState(GetResourceConfig("dashboards"), state)
			require.NoError(t, err)
			assert.Equal(t, tc.want, out.(*DashboardState).SerializedDashboard)
		})
	}
}
