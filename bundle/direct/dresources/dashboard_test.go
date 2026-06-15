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
	r := &ResourceDashboard{}
	state := &DashboardState{
		DashboardConfig: resources.DashboardConfig{
			DisplayName:         "test-dashboard",
			Etag:                "etag-123",
			SerializedDashboard: `{"pages":[{"name":"p1"}]}`,
		},
	}

	compacted, err := r.CompactState(state)
	require.NoError(t, err)

	// serialized_dashboard is replaced by a content hash; other fields are preserved.
	require.IsType(t, "", compacted.SerializedDashboard)
	assert.True(t, strings.HasPrefix(compacted.SerializedDashboard.(string), stateHashPrefix))
	assert.Equal(t, "test-dashboard", compacted.DisplayName)
	assert.Equal(t, "etag-123", compacted.Etag)

	// The original state is not mutated.
	assert.Equal(t, `{"pages":[{"name":"p1"}]}`, state.SerializedDashboard)

	// Compacting is idempotent.
	again, err := r.CompactState(compacted)
	require.NoError(t, err)
	assert.Equal(t, compacted.SerializedDashboard, again.SerializedDashboard)
}

// TestDashboardSerializedDashboardIsIgnoreRemoteChanges guards the SHA-only invariant:
// because serialized_dashboard is stored as a content hash, it must never be compared
// against the (full-content) remote value, i.e. it must be declared ignore_remote_changes.
func TestDashboardSerializedDashboardIsIgnoreRemoteChanges(t *testing.T) {
	cfg := GetResourceConfig("dashboards")
	path := structpath.NewStringKey(nil, "serialized_dashboard")

	found := false
	for _, rule := range cfg.IgnoreRemoteChanges {
		if path.HasPatternPrefix(rule.Field) {
			found = true
			break
		}
	}
	assert.True(t, found, "serialized_dashboard must be ignore_remote_changes for SHA-only state to be correct")
}
