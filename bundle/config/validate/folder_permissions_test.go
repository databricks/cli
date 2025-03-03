package validate

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFolderPermissionsInheritedWhenRootPathDoesNotExist(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Workspace/Users/foo@bar.com",
				ArtifactPath: "/Workspace/Users/otherfoo@bar.com/artifacts",
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
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users/otherfoo@bar.com/artifacts").Return(nil, &apierr.APIError{
		StatusCode: 404,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
	})
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users/otherfoo@bar.com").Return(nil, &apierr.APIError{
		StatusCode: 404,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
	})
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users/foo@bar.com").Return(nil, &apierr.APIError{
		StatusCode: 404,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
	})
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace/Users").Return(nil, &apierr.APIError{
		StatusCode: 404,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
	})
	api.EXPECT().GetStatusByPath(mock.Anything, "/Workspace").Return(&workspace.ObjectInfo{
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
	diags := ValidateFolderPermissions().Apply(context.Background(), b)
	require.Empty(t, diags)
}

func TestValidateFolderPermissionsFailsOnMissingBundlePermission(t *testing.T) {
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
	diags := ValidateFolderPermissions().Apply(context.Background(), b)
	require.Len(t, diags, 1)
	require.Equal(t, "untracked permissions apply to target workspace path", diags[0].Summary)
	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Equal(t, "The following permissions apply to the workspace folder at \"/Workspace/Users/foo@bar.com\" but are not configured in the bundle:\n- level: CAN_MANAGE, user_name: foo2@bar.com\n", diags[0].Detail)
}

func TestValidateFolderPermissionsFailsOnPermissionMismatch(t *testing.T) {
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
	diags := ValidateFolderPermissions().Apply(context.Background(), b)
	require.Len(t, diags, 1)
	require.Equal(t, "untracked permissions apply to target workspace path", diags[0].Summary)
	require.Equal(t, diag.Warning, diags[0].Severity)
}

func TestValidateFolderPermissionsFailsOnNoRootFolder(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/NotExisting",
				ArtifactPath: "/NotExisting/artifacts",
				FilePath:     "/NotExisting/files",
				StatePath:    "/NotExisting/state",
				ResourcePath: "/NotExisting/resources",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "foo@bar.com"},
			},
		},
	}
	m := mocks.NewMockWorkspaceClient(t)
	api := m.GetMockWorkspaceAPI()
	api.EXPECT().GetStatusByPath(mock.Anything, "/NotExisting").Return(nil, &apierr.APIError{
		StatusCode: 404,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
	})
	api.EXPECT().GetStatusByPath(mock.Anything, "/").Return(nil, &apierr.APIError{
		StatusCode: 404,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
	})

	b.SetWorkpaceClient(m.WorkspaceClient)
	diags := ValidateFolderPermissions().Apply(context.Background(), b)
	require.Len(t, diags, 1)
	require.Equal(t, "folder / and its parent folders do not exist", diags[0].Summary)
	require.Equal(t, diag.Error, diags[0].Severity)
}
