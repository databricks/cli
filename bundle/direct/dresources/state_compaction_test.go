package dresources

import (
	"crypto/sha256"
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
	// stateHashPlaceholderLen, so it is actually compacted.
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

// TestHashedFieldsAreTopLevel guards the shallow-copy assumption in CompactState:
// every hashed_fields path declared in resources.yml must be a top-level field. A nested
// path would be mutated through memory shared with the deploy value (see CompactState), so
// this fails CI the moment such a declaration is added instead of corrupting state at runtime.
func TestHashedFieldsAreTopLevel(t *testing.T) {
	for name, rc := range MustLoadConfig().Resources {
		for _, field := range rc.HashedFields {
			path, err := structpath.ParsePath(field)
			require.NoError(t, err, "%s: hashed_fields field %q", name, field)
			assert.Equal(t, 1, path.Len(), "%s: hashed_fields field %q must be a top-level field", name, field)
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

// TestHashStateValueIsValueAgnostic verifies hashStateValue hashes, not just
// dashboard JSON, since a hashed_fields field may hold arbitrary content (YAML, scripts, SQL).
// It also checks that small JSON values stay the same
func TestHashStateValueIsValueAgnostic(t *testing.T) {
	// A string encodes with two surrounding quotes, so a raw string of
	// stateHashPlaceholderLen-2 chars sits exactly at the limit; one more tips it over.
	atLimit := strings.Repeat("x", stateHashPlaceholderLen-2)

	cases := []struct {
		name    string
		content any
		hashed  bool
	}{
		{name: "yaml", content: "resources:\n  jobs:\n    nightly:\n      name: nightly-etl\n      tasks:\n        - task_key: main\n          notebook_task:\n            notebook_path: ./main.py\n", hashed: true},
		{name: "bash_script", content: "#!/usr/bin/env bash\nset -euo pipefail\nfor f in ./logs/*.log; do\n  echo \"processing $f\"\n  grep -c ERROR \"$f\" || true\ndone\n", hashed: true},
		{name: "python_script", content: "import sys\n\ndef main() -> None:\n    for line in sys.stdin:\n        print(line.rstrip().upper())\n\nif __name__ == \"__main__\":\n    main()\n", hashed: true},
		{name: "sql", content: "SELECT user_id, COUNT(*) AS n FROM events WHERE ts > current_date - INTERVAL 7 DAYS GROUP BY user_id ORDER BY n DESC LIMIT 100", hashed: true},
		{name: "markdown", content: "# Weekly report\n\nThis summarizes **activity** across all regions.\n\n- signups\n- active users\n- churn\n\nSee the appendix for methodology.\n", hashed: true},
		{name: "json_array", content: `[{"id":1,"tags":["a","b"]},{"id":2,"tags":["c","d"]},{"id":3,"tags":["e","f"]}]`, hashed: true},
		{name: "non_string_map", content: map[string]any{
			"query":  "SELECT * FROM t",
			"params": []any{"alpha", "beta", "gamma"},
			"limit":  100,
			"nested": map[string]any{"a": 1, "b": 2, "c": 3},
		}, hashed: true},
		{name: "small_stays_raw", content: smallDashboard, hashed: false},
		{name: "at_size_limit", content: atLimit, hashed: false},
		{name: "over_size_limit", content: atLimit + "x", hashed: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := hashStateValue(tc.content)
			require.NoError(t, err)

			if tc.hashed {
				require.IsType(t, "", out)
				assert.True(t, isStateHashPlaceholder(out.(string)))
				assert.Len(t, out, stateHashPlaceholderLen)
			} else {
				assert.Equal(t, tc.content, out)
			}

			again, err := hashStateValue(tc.content)
			require.NoError(t, err)
			assert.Equal(t, out, again)

			rehashed, err := hashStateValue(out)
			require.NoError(t, err)
			assert.Equal(t, out, rehashed)
		})
	}
}

// TestIsStateHashPlaceholder verifies the guard matches ^sha256:[a-f0-9]{64}$ exactly,
// so a string that merely shares the prefix is not treated as already-hashed.
func TestIsStateHashPlaceholder(t *testing.T) {
	hex64 := strings.Repeat("a", sha256.Size*2)
	assert.True(t, isStateHashPlaceholder(stateHashPrefix+hex64))

	for name, s := range map[string]string{
		"prefix_only":  stateHashPrefix,
		"too_short":    stateHashPrefix + strings.Repeat("a", 63),
		"too_long":     stateHashPrefix + strings.Repeat("a", 65),
		"uppercase":    stateHashPrefix + strings.Repeat("A", 64),
		"non_hex":      stateHashPrefix + strings.Repeat("g", 64),
		"no_prefix":    hex64,
		"empty":        "",
		"content_like": `{"pages":[]}`,
	} {
		t.Run(name, func(t *testing.T) {
			assert.False(t, isStateHashPlaceholder(s))
		})
	}
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
