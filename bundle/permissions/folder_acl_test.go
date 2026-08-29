package permissions

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func user(name string, groups ...string) *iam.User {
	u := &iam.User{UserName: name}
	for _, g := range groups {
		u.Groups = append(u.Groups, iam.ComplexValue{Display: g})
	}
	return u
}

func aclFor(t *testing.T, acl []workspace.WorkspaceObjectAccessControlResponse) *FolderACL {
	t.Helper()
	return &FolderACL{
		RequestedPath: "/Workspace/f",
		ResolvedPath:  "/Workspace/f",
		Permissions:   ObjectAclToResourcePermissions("/Workspace/f", acl),
	}
}

func TestFolderACLCanWrite(t *testing.T) {
	editUser := []workspace.WorkspaceObjectAccessControlResponse{
		{
			UserName: "me@example.com",
			AllPermissions: []workspace.WorkspaceObjectPermission{
				{PermissionLevel: "CAN_EDIT"},
			},
		},
	}
	manageGroup := []workspace.WorkspaceObjectAccessControlResponse{
		{
			GroupName: "devs",
			AllPermissions: []workspace.WorkspaceObjectPermission{
				{PermissionLevel: "CAN_MANAGE"},
			},
		},
	}
	readOnlyUser := []workspace.WorkspaceObjectAccessControlResponse{
		{
			UserName: "me@example.com",
			AllPermissions: []workspace.WorkspaceObjectPermission{
				{PermissionLevel: "CAN_READ"},
			},
		},
	}
	sp := []workspace.WorkspaceObjectAccessControlResponse{
		{
			ServicePrincipalName: "sp-uuid",
			AllPermissions: []workspace.WorkspaceObjectPermission{
				{PermissionLevel: "CAN_MANAGE"},
			},
		},
	}

	tests := []struct {
		name string
		acl  []workspace.WorkspaceObjectAccessControlResponse
		user *iam.User
		want bool
	}{
		{"direct write", editUser, user("me@example.com"), true},
		{"write via group", manageGroup, user("me@example.com", "devs"), true},
		{"read-only is not writable", readOnlyUser, user("me@example.com"), false},
		{"no matching entry", editUser, user("other@example.com"), false},
		{"service principal write", sp, user("sp-uuid"), true},

		// Conservative cases: assume writable when it cannot be ruled out.
		{"admin bypasses folder ACL", readOnlyUser, user("me@example.com", "admins"), true},
		{"write via group not in user's known groups", manageGroup, user("me@example.com"), false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, aclFor(t, tc.acl).CanWrite(tc.user))
		})
	}
}

// These ACLs mirror the shape of real workspace folder permissions (verified
// against live traffic): a home-style folder granting the owner CAN_MANAGE
// directly plus inherited admin/service-principal entries, and a shared folder
// granting CAN_MANAGE to a group the user belongs to rather than to the user
// directly.

// homeDirACL: owner has direct CAN_MANAGE (writable).
var homeDirACL = []workspace.WorkspaceObjectAccessControlResponse{
	{
		UserName: "owner@example.com",
		AllPermissions: []workspace.WorkspaceObjectPermission{
			{PermissionLevel: "CAN_MANAGE"},
		},
	},
	{
		ServicePrincipalName: "00000000-0000-0000-0000-000000000000",
		AllPermissions: []workspace.WorkspaceObjectPermission{
			{PermissionLevel: "CAN_MANAGE", Inherited: true},
		},
	},
	{
		GroupName: "admins",
		AllPermissions: []workspace.WorkspaceObjectPermission{
			{PermissionLevel: "CAN_MANAGE", Inherited: true},
		},
	},
}

// sharedDirACL: CAN_MANAGE granted to the "users" group (write via group, not a
// direct grant).
var sharedDirACL = []workspace.WorkspaceObjectAccessControlResponse{
	{
		GroupName: "users",
		AllPermissions: []workspace.WorkspaceObjectPermission{
			{PermissionLevel: "CAN_MANAGE"},
		},
	},
	{
		GroupName: "admins",
		AllPermissions: []workspace.WorkspaceObjectPermission{
			{PermissionLevel: "CAN_MANAGE", Inherited: true},
		},
	},
}

func TestFolderACLCanWriteRealisticACLs(t *testing.T) {
	me := user("owner@example.com", "engineering", "users")

	assert.True(t, aclFor(t, homeDirACL).CanWrite(me), "own home dir: direct CAN_MANAGE")
	assert.True(t, aclFor(t, sharedDirACL).CanWrite(me), "shared dir: CAN_MANAGE via users group")

	// A different user without any of these grants cannot write.
	other := user("someone.else@example.com")
	assert.False(t, aclFor(t, homeDirACL).CanWrite(other))
	assert.False(t, aclFor(t, sharedDirACL).CanWrite(other))
}

func TestCheckWritable(t *testing.T) {
	const folder = "/Workspace/f"
	me := user("me@example.com")

	writableACL := &workspace.WorkspaceObjectPermissions{
		AccessControlList: []workspace.WorkspaceObjectAccessControlResponse{
			{
				UserName: "me@example.com",
				AllPermissions: []workspace.WorkspaceObjectPermission{
					{PermissionLevel: "CAN_MANAGE"},
				},
			},
		},
	}
	readOnlyACL := &workspace.WorkspaceObjectPermissions{
		AccessControlList: []workspace.WorkspaceObjectAccessControlResponse{
			{
				UserName: "me@example.com",
				AllPermissions: []workspace.WorkspaceObjectPermission{
					{PermissionLevel: "CAN_READ"},
				},
			},
		},
	}

	tests := []struct {
		name     string
		getPerms func() (*workspace.WorkspaceObjectPermissions, error)
		want     Writability
	}{
		{
			"writable",
			func() (*workspace.WorkspaceObjectPermissions, error) { return writableACL, nil },
			WritabilityWritable,
		},
		{
			// Reading the ACL is denied because it requires manage access, so the
			// user definitely cannot write. This is the common real-world case.
			"permission denied on ACL read means not writable",
			func() (*workspace.WorkspaceObjectPermissions, error) {
				return nil, &apierr.APIError{StatusCode: 403, ErrorCode: "PERMISSION_DENIED"}
			},
			WritabilityNotWritable,
		},
		{
			"readable ACL without a write grant means not writable",
			func() (*workspace.WorkspaceObjectPermissions, error) { return readOnlyACL, nil },
			WritabilityNotWritable,
		},
		{
			"other errors are unknown",
			func() (*workspace.WorkspaceObjectPermissions, error) {
				return nil, &apierr.APIError{StatusCode: 500}
			},
			WritabilityUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := mocks.NewMockWorkspaceClient(t)
			w := m.GetMockWorkspaceAPI()
			w.EXPECT().GetStatusByPath(mock.Anything, folder).
				Return(&workspace.ObjectInfo{ObjectId: 7, Path: folder}, nil)
			w.EXPECT().GetPermissions(mock.Anything, mock.Anything).Return(tc.getPerms())

			assert.Equal(t, tc.want, CheckWritable(t.Context(), w, folder, me))
		})
	}
}

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
	assert.True(t, acl.CanWrite(user("me@example.com")))
}
