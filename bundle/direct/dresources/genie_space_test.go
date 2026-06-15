package dresources

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsMissingGenieParentPathError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "standard missing error",
			err: &apierr.APIError{
				StatusCode: 404,
				ErrorCode:  "NOT_FOUND",
				Message:    "not found",
			},
			want: true,
		},
		{
			name: "invalid parameter tree node missing error",
			err: &apierr.APIError{
				StatusCode: 400,
				ErrorCode:  "INVALID_PARAMETER_VALUE",
				Message:    "NOT_FOUND: Tree node with path /Workspace/foo does not exist",
			},
			want: true,
		},
		{
			name: "other invalid parameter error",
			err: &apierr.APIError{
				StatusCode: 400,
				ErrorCode:  "INVALID_PARAMETER_VALUE",
				Message:    "some other validation failure",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isMissingGenieParentPathError(tt.err))
		})
	}
}

func TestGenieSpaceDoCreateRetriesWhenParentPathLooksMissing(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	r := (&ResourceGenieSpace{}).New(m.WorkspaceClient)

	req := dashboards.GenieCreateSpaceRequest{
		Title:           "test genie space",
		Description:     "test description",
		ParentPath:      "/Workspace/test-parent",
		WarehouseId:     "test-warehouse-id",
		SerializedSpace: "{}",
	}

	m.GetMockGenieAPI().EXPECT().
		CreateSpace(ctx, req).
		Return(nil, &apierr.APIError{
			StatusCode: 400,
			ErrorCode:  "INVALID_PARAMETER_VALUE",
			Message:    "NOT_FOUND: Tree node with path /Workspace/test-parent does not exist",
		}).
		Once()

	m.GetMockWorkspaceAPI().EXPECT().
		MkdirsByPath(ctx, "/Workspace/test-parent").
		Return(nil).
		Once()

	m.GetMockGenieAPI().EXPECT().
		CreateSpace(ctx, req).
		Return(&dashboards.GenieSpace{
			SpaceId:         "space-id",
			Title:           "test genie space",
			Description:     "test description",
			WarehouseId:     "test-warehouse-id",
			SerializedSpace: "{}",
		}, nil).
		Once()

	id, state, err := r.DoCreate(ctx, &resources.GenieSpaceConfig{
		Title:           "test genie space",
		Description:     "test description",
		ParentPath:      "/Workspace/test-parent",
		WarehouseId:     "test-warehouse-id",
		SerializedSpace: "{}",
	})
	require.NoError(t, err)
	assert.Equal(t, "space-id", id)
	require.NotNil(t, state)
	assert.Equal(t, "test genie space", state.Title)
}

func TestGenieSpaceDoUpdateRoundTripsEtag(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	r := (&ResourceGenieSpace{}).New(m.WorkspaceClient)

	entry := &deployplan.PlanEntry{
		Changes: deployplan.Changes{
			"title": {Action: deployplan.Update, Old: "old", New: "new"},
		},
	}

	// The stored etag (etag-7) must NOT be sent as an If-Match guard — it would
	// 409 after a backend serialized_space schema migration. Only the etag from
	// the response is persisted, for drift detection on the next plan.
	m.GetMockGenieAPI().EXPECT().
		UpdateSpace(ctx, dashboards.GenieUpdateSpaceRequest{
			SpaceId: "space-id",
			Title:   "new",
		}).
		Return(&dashboards.GenieSpace{
			SpaceId: "space-id",
			Title:   "new",
			Etag:    "etag-8",
		}, nil).
		Once()

	state, err := r.DoUpdate(ctx, "space-id", &resources.GenieSpaceConfig{
		Title: "new",
		Etag:  "etag-7",
	}, entry)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, "etag-8", state.Etag)
}

func TestGenieSpaceDoUpdateAlwaysSendsSerializedSpace(t *testing.T) {
	ctx := t.Context()
	m := mocks.NewMockWorkspaceClient(t)
	r := (&ResourceGenieSpace{}).New(m.WorkspaceClient)

	// Even though the plan entry only marks an unrelated field changed, the body
	// is sent so the deploy converges the space to the bundle config.
	entry := &deployplan.PlanEntry{
		Changes: deployplan.Changes{
			"title": {Action: deployplan.Update, Old: "old", New: "new"},
		},
	}

	m.GetMockGenieAPI().EXPECT().
		UpdateSpace(ctx, dashboards.GenieUpdateSpaceRequest{
			SpaceId:         "space-id",
			Title:           "new",
			SerializedSpace: "{\"converge\":\"me\"}",
		}).
		Return(&dashboards.GenieSpace{
			SpaceId:         "space-id",
			Title:           "new",
			SerializedSpace: "{\"converge\":\"me\"}",
		}, nil).
		Once()

	state, err := r.DoUpdate(ctx, "space-id", &resources.GenieSpaceConfig{
		Title:           "new",
		SerializedSpace: "{\"converge\":\"me\"}",
	}, entry)
	require.NoError(t, err)
	require.NotNil(t, state)
	assert.Equal(t, "{\"converge\":\"me\"}", state.SerializedSpace)
}

func TestGenieSpaceOverrideChangeDescEtag(t *testing.T) {
	r := &ResourceGenieSpace{}
	etagPath := structpath.MustParsePath("etag")

	t.Run("Skip when stored matches remote", func(t *testing.T) {
		change := &ChangeDesc{Old: "etag-7", Remote: "etag-7"}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), etagPath, change, nil))
		assert.Equal(t, deployplan.Skip, change.Action)
	})

	t.Run("Update when stored differs from remote", func(t *testing.T) {
		change := &ChangeDesc{Old: "etag-7", Remote: "etag-8"}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), etagPath, change, nil))
		assert.Equal(t, deployplan.Update, change.Action)
	})

	t.Run("Other paths are untouched", func(t *testing.T) {
		titlePath := structpath.MustParsePath("title")
		change := &ChangeDesc{Action: deployplan.Update, Old: "a", Remote: "b"}
		require.NoError(t, r.OverrideChangeDesc(t.Context(), titlePath, change, nil))
		assert.Equal(t, deployplan.Update, change.Action)
	})
}

func TestGenieSpaceCompactState(t *testing.T) {
	r := &ResourceGenieSpace{}
	state := &resources.GenieSpaceConfig{
		Title:           "test-space",
		Etag:            "etag-7",
		SerializedSpace: `{"datasets":[{"name":"d1"}]}`,
	}

	compacted, err := r.CompactState(state)
	require.NoError(t, err)

	require.IsType(t, "", compacted.SerializedSpace)
	assert.True(t, strings.HasPrefix(compacted.SerializedSpace.(string), stateHashPrefix))
	assert.Equal(t, "test-space", compacted.Title)
	assert.Equal(t, "etag-7", compacted.Etag)

	// The original state is not mutated.
	assert.Equal(t, `{"datasets":[{"name":"d1"}]}`, state.SerializedSpace)
}

// TestGenieSpaceSerializedSpaceIsIgnoreRemoteChanges guards the SHA-only invariant:
// serialized_space is stored as a content hash, so it must never be compared against
// the remote value, i.e. it must be declared ignore_remote_changes.
func TestGenieSpaceSerializedSpaceIsIgnoreRemoteChanges(t *testing.T) {
	cfg := GetResourceConfig("genie_spaces")
	path := structpath.NewStringKey(nil, "serialized_space")

	found := false
	for _, rule := range cfg.IgnoreRemoteChanges {
		if path.HasPatternPrefix(rule.Field) {
			found = true
			break
		}
	}
	assert.True(t, found, "serialized_space must be ignore_remote_changes for SHA-only state to be correct")
}
