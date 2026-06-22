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

func TestDashboardState_JSONSerialization_PublishedField(t *testing.T) {
	state := &DashboardState{
		DashboardConfig: resources.DashboardConfig{
			DisplayName: "test-dashboard",
			WarehouseId: "warehouse123",
		},
		Published: true,
	}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"published":true`)
}

func TestDashboardCompactState(t *testing.T) {
	state := &DashboardState{
		DashboardConfig: resources.DashboardConfig{
			DisplayName:         "test-dashboard",
			Etag:                "etag-123",
			SerializedDashboard: `{"pages":[{"name":"p1"}]}`,
		},
	}

	out, err := CompactState(GetResourceConfig("dashboards"), state)
	require.NoError(t, err)
	compacted := out.(*DashboardState)

	// serialized_dashboard is replaced by a content hash; other fields are preserved.
	require.IsType(t, "", compacted.SerializedDashboard)
	assert.True(t, strings.HasPrefix(compacted.SerializedDashboard.(string), stateHashPrefix))
	assert.Equal(t, "test-dashboard", compacted.DisplayName)
	assert.Equal(t, "etag-123", compacted.Etag)

	// The original state is not mutated.
	assert.Equal(t, `{"pages":[{"name":"p1"}]}`, state.SerializedDashboard)

	// Compacting is idempotent.
	out2, err := CompactState(GetResourceConfig("dashboards"), compacted)
	require.NoError(t, err)
	assert.Equal(t, compacted.SerializedDashboard, out2.(*DashboardState).SerializedDashboard)
}

// TestDashboardSerializedDashboardStateRules guards the SHA-only invariant. The field
// must be declared hashed_in_state (so it is persisted as a hash) and, because the hash
// can never equal the full-content remote value, it must also be ignore_remote_changes.
func TestDashboardSerializedDashboardStateRules(t *testing.T) {
	cfg := GetResourceConfig("dashboards")
	path := structpath.NewStringKey(nil, "serialized_dashboard")

	hasRule := func(rules []FieldRule) bool {
		for _, rule := range rules {
			if path.HasPatternPrefix(rule.Field) {
				return true
			}
		}
		return false
	}

	assert.True(t, hasRule(cfg.HashedInState), "serialized_dashboard must be declared hashed_in_state")
	assert.True(t, hasRule(cfg.IgnoreRemoteChanges), "serialized_dashboard must be ignore_remote_changes for SHA-only state to be correct")
}
