package validate

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestValidateFolderPermissions(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Workspace/Users/foo@bar.com",
				ArtifactPath: "/Workspace/Users/foo@bar.com/artifacts",
				FilePath:     "/Workspace/Users/foo@bar.com/files",
				StatePath:    "/Workspace/Users/foo@bar.com/state",
				ResourcePath: "/Workspace/Users/foo@bar.com/resources",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "foo@bar.com"},
			},
		},
	}
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockWorkspaceAPI()
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users/foo@bar.com").Return(&workspace.ObjectInfo{
		ObjectId: 1234,
	}, nil)

	api.EXPECT().GetPermissions(mock.Anything, workspace.GetWorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   "1234",
		WorkspaceObjectType: "directories",
	}).Return(&workspace.WorkspaceObjectPermissions{
		ObjectId: "1234",
		AccessControlList: []workspace.WorkspaceObjectAccessControlResponse{
			{
				UserName: "foo@bar.com",
				AllPermissions: []workspace.WorkspaceObjectPermission{
					{PermissionLevel: "CAN_MANAGE"},
				},
			},
		},
	}, nil)

	b.SetWorkpaceClient(m.WorkspaceClient)
	rb := bundle.ReadOnly(b)

	diags := bundle.ApplyReadOnly(context.Background(), rb, ValidateFolderPermissions())
	require.Empty(t, diags)
}

func TestValidateFolderPermissionsDifferentCount(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Workspace/Users/foo@bar.com",
				ArtifactPath: "/Workspace/Users/foo@bar.com/artifacts",
				FilePath:     "/Workspace/Users/foo@bar.com/files",
				StatePath:    "/Workspace/Users/foo@bar.com/state",
				ResourcePath: "/Workspace/Users/foo@bar.com/resources",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "foo@bar.com"},
			},
		},
	}
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockWorkspaceAPI()
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users/foo@bar.com").Return(&workspace.ObjectInfo{
		ObjectId: 1234,
	}, nil)

	api.EXPECT().GetPermissions(mock.Anything, workspace.GetWorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   "1234",
		WorkspaceObjectType: "directories",
	}).Return(&workspace.WorkspaceObjectPermissions{
		ObjectId: "1234",
		AccessControlList: []workspace.WorkspaceObjectAccessControlResponse{
			{
				UserName: "foo@bar.com",
				AllPermissions: []workspace.WorkspaceObjectPermission{
					{PermissionLevel: "CAN_MANAGE"},
				},
			},
			{
				UserName: "foo2@bar.com",
				AllPermissions: []workspace.WorkspaceObjectPermission{
					{PermissionLevel: "CAN_MANAGE"},
				},
			},
		},
	}, nil)

	b.SetWorkpaceClient(m.WorkspaceClient)
	rb := bundle.ReadOnly(b)

	diags := bundle.ApplyReadOnly(context.Background(), rb, ValidateFolderPermissions())
	require.Len(t, diags, 1)
	require.Equal(t, "permissions count mismatch", diags[0].Summary)
}

func TestValidateFolderPermissionsDifferentPermission(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Workspace/Users/foo@bar.com",
				ArtifactPath: "/Workspace/Users/foo@bar.com/artifacts",
				FilePath:     "/Workspace/Users/foo@bar.com/files",
				StatePath:    "/Workspace/Users/foo@bar.com/state",
				ResourcePath: "/Workspace/Users/foo@bar.com/resources",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "foo@bar.com"},
			},
		},
	}
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockWorkspaceAPI()
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users/foo@bar.com").Return(&workspace.ObjectInfo{
		ObjectId: 1234,
	}, nil)

	api.EXPECT().GetPermissions(mock.Anything, workspace.GetWorkspaceObjectPermissionsRequest{
		WorkspaceObjectId:   "1234",
		WorkspaceObjectType: "directories",
	}).Return(&workspace.WorkspaceObjectPermissions{
		ObjectId: "1234",
		AccessControlList: []workspace.WorkspaceObjectAccessControlResponse{
			{
				UserName: "foo2@bar.com",
				AllPermissions: []workspace.WorkspaceObjectPermission{
					{PermissionLevel: "CAN_MANAGE"},
				},
			},
		},
	}, nil)

	b.SetWorkpaceClient(m.WorkspaceClient)
	rb := bundle.ReadOnly(b)

	diags := bundle.ApplyReadOnly(context.Background(), rb, ValidateFolderPermissions())
	require.Len(t, diags, 1)
	require.Equal(t, "permission not found", diags[0].Summary)
}
