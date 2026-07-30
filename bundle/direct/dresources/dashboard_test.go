package dresources

import (
	"encoding/json"
	"slices"
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

// TestDashboardSerializedDashboardStateRules documents that serialized_dashboard carries
// two independent declarations: hashed_in_state (persist only its hash, since the blob is
// large) and ignore_remote_changes (the server normalizes the content, so its remote hash
// never matches the config hash — drift is detected via etag instead). The two are
// orthogonal in general; serialized_dashboard just happens to need both.
func TestDashboardSerializedDashboardStateRules(t *testing.T) {
	cfg := GetResourceConfig("dashboards")
	path := structpath.NewStringKey(nil, "serialized_dashboard")

	ignoresRemote := false
	for _, rule := range cfg.IgnoreRemoteChanges {
		if path.HasPatternPrefix(rule.Field) {
			ignoresRemote = true
			break
		}
	}

	assert.True(t, slices.Contains(cfg.HashedInState, "serialized_dashboard"), "serialized_dashboard must be declared hashed_in_state")
	assert.True(t, ignoresRemote, "serialized_dashboard must be ignore_remote_changes (server normalizes the content)")
}
