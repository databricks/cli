package dresources

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	registerTestConfig("dashboards", &resources.Dashboard{
		DisplayName:         "my-dashboard",
		ParentPath:          "/Workspace/Users/user@example.com",
		WarehouseId:         "test-warehouse-id",
		SerializedDashboard: `{"pages":[{"name":"page1","displayName":"Page 1","pageType":"PAGE_TYPE_CANVAS"}]}`,

		DatasetCatalog: "main",
		DatasetSchema:  "myschema",
	})

	registerTestDeps("dashboards.permissions", func(ctx context.Context, client *databricks.WorkspaceClient) (any, error) {
		parentPath := "/Workspace/Users/user@example.com"

		// Create parent directory if it doesn't exist
		err := client.Workspace.MkdirsByPath(ctx, parentPath)
		if err != nil {
			return nil, err
		}

		resp, err := client.Lakeview.Create(ctx, dashboards.CreateDashboardRequest{
			Dashboard: dashboards.Dashboard{
				DisplayName:         "dashboard-permissions",
				ParentPath:          parentPath,
				SerializedDashboard: `{"pages":[{"name":"page1","displayName":"Page 1"}]}`,
			},
		})
		if err != nil {
			return nil, err
		}

		return &PermissionsState{
			ObjectID: "/dashboards/" + resp.DashboardId,
			EmbeddedSlice: []StatePermission{{
				Level:    "CAN_MANAGE",
				UserName: "user@example.com",
			}},
		}, nil
	})

	registerKnownMissingInStateType("dashboards",
		"file_path",
	)
}

func TestDashboardState_JSONSerialization_PublishedField(t *testing.T) {
	state := &DashboardState{
		DisplayName: "test-dashboard",
		WarehouseId: "warehouse123",
		Published:   true,
	}

	data, err := json.Marshal(state)
	require.NoError(t, err)

	assert.Contains(t, string(data), `"published":true`)
}

func TestDashboardCompactState(t *testing.T) {
	requireLargeEnoughToHash(t, largeDashboard)

	state := &DashboardState{
		DisplayName:         "test-dashboard",
		Etag:                "etag-123",
		SerializedDashboard: largeDashboard,
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
	assert.Equal(t, largeDashboard, state.SerializedDashboard)

	// Compacting is idempotent.
	out2, err := CompactState(GetResourceConfig("dashboards"), compacted)
	require.NoError(t, err)
	assert.Equal(t, compacted.SerializedDashboard, out2.(*DashboardState).SerializedDashboard)
}

// TestDashboardSerializedDashboardStateRules documents that serialized_dashboard carries
// two independent declarations: hashed_fields (persist only its hash, since the blob is
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

	assert.True(t, slices.Contains(cfg.HashedFields, "serialized_dashboard"), "serialized_dashboard must be declared hashed_fields")
	assert.True(t, ignoresRemote, "serialized_dashboard must be ignore_remote_changes (server normalizes the content)")
}
