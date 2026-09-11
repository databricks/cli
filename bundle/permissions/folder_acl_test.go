package permissions

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestResolveFolderACLWalksUpToClosestExistingAncestor(t *testing.T) {
	m := mocks.NewMockWorkspaceClient(t)
	w := m.GetMockWorkspaceAPI()

	// The requested folder does not exist yet; its parent does.
	w.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/root/files").
		Return(nil, &apierr.APIError{StatusCode: 404})
	w.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/root").
		Return(&workspace.ObjectInfo{ObjectId: 42, Path: "/Workspace/root"}, nil)
	w.EXPECT().GetPermissions(mock.Anything, workspace.GetWorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   "42",
		WorkspaceObjectType: "directories",
	}).Return(&workspace.WorkspaceObjectPermissions{
		AccessControlList: []workspace.WorkspaceObjectAccessControlResponse{
			{
				UserName: "me@example.com",
				AllPermissions: []workspace.WorkspaceObjectPermission{
					{PermissionLevel: "CAN_MANAGE"},
				},
			},
		},
	}, nil)

	acl, err := ResolveFolderACL(t.Context(), w, "/Workspace/root/files")
	require.NoError(t, err)

	// Resolved via the ancestor, but reported against the requested path.
	assert.Equal(t, "/Workspace/root/files", acl.RequestedPath)
	assert.Equal(t, "/Workspace/root", acl.ResolvedPath)
	assert.Equal(t, "/Workspace/root/files", acl.Permissions.Path)
}
