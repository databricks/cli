package dresources

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
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
