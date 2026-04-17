package workspace

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetWorkspaceContentDir(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	m.GetMockCurrentUserAPI().EXPECT().Me(ctx).Return(&iam.User{
		UserName: "testuser@example.com",
	}, nil)

	dir, err := GetWorkspaceContentDir(ctx, m.WorkspaceClient, "0.1.0", "cluster-123")
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/Users/testuser@example.com/.databricks/ssh-tunnel/0.1.0/cluster-123", dir)
}

func TestGetWorkspaceVersionedDir(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	m.GetMockCurrentUserAPI().EXPECT().Me(ctx).Return(&iam.User{
		UserName: "testuser@example.com",
	}, nil)

	dir, err := GetWorkspaceVersionedDir(ctx, m.WorkspaceClient, "0.2.0")
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/Users/testuser@example.com/.databricks/ssh-tunnel/0.2.0", dir)
}

func TestGetWorkspaceContentDir_ServerlessSession(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	m.GetMockCurrentUserAPI().EXPECT().Me(ctx).Return(&iam.User{
		UserName: "testuser@example.com",
	}, nil)

	dir, err := GetWorkspaceContentDir(ctx, m.WorkspaceClient, "0.1.0", "databricks-gpu-1xa10-961dabbd")
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/Users/testuser@example.com/.databricks/ssh-tunnel/0.1.0/databricks-gpu-1xa10-961dabbd", dir)
}

func TestGetWorkspaceContentDir_CurrentUserError(t *testing.T) {
	ctx := cmdio.MockDiscard(t.Context())
	m := mocks.NewMockWorkspaceClient(t)

	m.GetMockCurrentUserAPI().EXPECT().Me(ctx).Return(nil, assert.AnError)

	_, err := GetWorkspaceContentDir(ctx, m.WorkspaceClient, "0.1.0", "cluster-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current user")
}

func TestWorkspaceMetadata_JSON(t *testing.T) {
	tests := []struct {
		name     string
		metadata WorkspaceMetadata
		json     string
	}{
		{
			name:     "with cluster ID",
			metadata: WorkspaceMetadata{Port: 2222, ClusterID: "abc-123"},
			json:     `{"port":2222,"cluster_id":"abc-123"}`,
		},
		{
			name:     "without cluster ID",
			metadata: WorkspaceMetadata{Port: 3333},
			json:     `{"port":3333}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Marshal
			data, err := json.Marshal(tt.metadata)
			require.NoError(t, err)
			assert.JSONEq(t, tt.json, string(data))

			// Unmarshal
			var parsed WorkspaceMetadata
			err = json.Unmarshal([]byte(tt.json), &parsed)
			require.NoError(t, err)
			assert.Equal(t, tt.metadata, parsed)
		})
	}
}

func TestWorkspaceMetadata_InvalidJSON(t *testing.T) {
	var metadata WorkspaceMetadata
	err := json.Unmarshal([]byte("not json"), &metadata)
	assert.Error(t, err)
}
