package permissions

import (
	"github.com/databricks/cli/libs/sdktypes/scim/v1"
	"testing"

	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/cli/libs/workspaceapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func user(name string, groups ...string) *scim.User {
	u := &scim.User{UserName: new(name)}
	for _, g := range groups {
		u.Groups = append(u.Groups, scim.ComplexValue{Display: new(g)})
	}
	return u
}

func aclFor(t *testing.T, acl []workspaceapi.WorkspaceObjectAccessControlResponse) *FolderACL {
	t.Helper()
	return &FolderACL{
		RequestedPath: "/Workspace/f",
		ResolvedPath:  "/Workspace/f",
		Permissions:   ObjectAclToResourcePermissions("/Workspace/f", acl),
	}
}

func TestFolderACLCanWrite(t *testing.T) {
	editUser := []workspaceapi.WorkspaceObjectAccessControlResponse{
		{
			UserName: "me@example.com",
			AllPermissions: []workspaceapi.WorkspaceObjectPermission{
				{PermissionLevel: "CAN_EDIT"},
			},
		},
	}
	manageGroup := []workspaceapi.WorkspaceObjectAccessControlResponse{
		{
			GroupName: "devs",
			AllPermissions: []workspaceapi.WorkspaceObjectPermission{
				{PermissionLevel: "CAN_MANAGE"},
			},
		},
	}
	readOnlyUser := []workspaceapi.WorkspaceObjectAccessControlResponse{
		{
			UserName: "me@example.com",
			AllPermissions: []workspaceapi.WorkspaceObjectPermission{
				{PermissionLevel: "CAN_READ"},
			},
		},
	}
	sp := []workspaceapi.WorkspaceObjectAccessControlResponse{
		{
			ServicePrincipalName: "sp-uuid",
			AllPermissions: []workspaceapi.WorkspaceObjectPermission{
				{PermissionLevel: "CAN_MANAGE"},
			},
		},
	}

	tests := []struct {
		name string
		acl  []workspaceapi.WorkspaceObjectAccessControlResponse
		user *scim.User
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
var homeDirACL = []workspaceapi.WorkspaceObjectAccessControlResponse{
	{
		UserName: "owner@example.com",
		AllPermissions: []workspaceapi.WorkspaceObjectPermission{
			{PermissionLevel: "CAN_MANAGE"},
		},
	},
	{
		ServicePrincipalName: "00000000-0000-0000-0000-000000000000",
		AllPermissions: []workspaceapi.WorkspaceObjectPermission{
			{PermissionLevel: "CAN_MANAGE", Inherited: true},
		},
	},
	{
		GroupName: "admins",
		AllPermissions: []workspaceapi.WorkspaceObjectPermission{
			{PermissionLevel: "CAN_MANAGE", Inherited: true},
		},
	},
}

// sharedDirACL: CAN_MANAGE granted to the "users" group (write via group, not a
// direct grant).
var sharedDirACL = []workspaceapi.WorkspaceObjectAccessControlResponse{
	{
		GroupName: "users",
		AllPermissions: []workspaceapi.WorkspaceObjectPermission{
			{PermissionLevel: "CAN_MANAGE"},
		},
	},
	{
		GroupName: "admins",
		AllPermissions: []workspaceapi.WorkspaceObjectPermission{
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

// workspaceWithACL serves get-status for the given directories (path -> object id)
// and returns getPerms for the permissions read of any of them.
func workspaceWithACL(t *testing.T, ids map[string]int64, getPerms func() any) *workspaceapi.Client {
	w, server := bundletest.NewClients(t)
	server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
		path := req.URL.Query().Get("path")
		id, ok := ids[path]
		if !ok {
			return testserver.Response{StatusCode: 404, Body: map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": path}}
		}
		return workspaceapi.ObjectInfo{ObjectId: id, ObjectType: workspaceapi.ObjectTypeDirectory, Path: path}
	})
	server.Handle("GET", "/api/2.0/permissions/directories/{id}", func(req testserver.Request) any {
		return getPerms()
	})
	return w.Workspace
}

func apiError(status int, code string) testserver.Response {
	return testserver.Response{StatusCode: status, Body: map[string]string{"error_code": code, "message": code}}
}

func TestCheckWritable(t *testing.T) {
	const folder = "/Workspace/f"
	me := user("me@example.com")

	writableACL := &workspaceapi.WorkspaceObjectPermissions{
		AccessControlList: []workspaceapi.WorkspaceObjectAccessControlResponse{
			{
				UserName: "me@example.com",
				AllPermissions: []workspaceapi.WorkspaceObjectPermission{
					{PermissionLevel: "CAN_MANAGE"},
				},
			},
		},
	}
	readOnlyACL := &workspaceapi.WorkspaceObjectPermissions{
		AccessControlList: []workspaceapi.WorkspaceObjectAccessControlResponse{
			{
				UserName: "me@example.com",
				AllPermissions: []workspaceapi.WorkspaceObjectPermission{
					{PermissionLevel: "CAN_READ"},
				},
			},
		},
	}

	tests := []struct {
		name     string
		getPerms func() any
		want     Writability
	}{
		{
			"writable",
			func() any { return writableACL },
			WritabilityWritable,
		},
		{
			// Reading the ACL is denied because it requires manage access, so the
			// user definitely cannot write. This is the common real-world case.
			"permission denied on ACL read means not writable",
			func() any { return apiError(403, "PERMISSION_DENIED") },
			WritabilityNotWritable,
		},
		{
			"readable ACL without a write grant means not writable",
			func() any { return readOnlyACL },
			WritabilityNotWritable,
		},
		{
			"other errors are unknown",
			func() any { return apiError(500, "INTERNAL_ERROR") },
			WritabilityUnknown,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w := workspaceWithACL(t, map[string]int64{folder: 7}, tc.getPerms)
			assert.Equal(t, tc.want, CheckWritable(t.Context(), w, folder, me))
		})
	}
}

func TestResolveFolderACLWalksUpToClosestExistingAncestor(t *testing.T) {
	// The requested folder does not exist yet; its parent does.
	w := workspaceWithACL(t, map[string]int64{"/Workspace/root": 42}, func() any {
		return &workspaceapi.WorkspaceObjectPermissions{
			AccessControlList: []workspaceapi.WorkspaceObjectAccessControlResponse{
				{
					UserName: "me@example.com",
					AllPermissions: []workspaceapi.WorkspaceObjectPermission{
						{PermissionLevel: "CAN_MANAGE"},
					},
				},
			},
		}
	})

	acl, err := ResolveFolderACL(t.Context(), w, "/Workspace/root/files")
	require.NoError(t, err)

	// Resolved via the ancestor, but reported against the requested path.
	assert.Equal(t, "/Workspace/root/files", acl.RequestedPath)
	assert.Equal(t, "/Workspace/root", acl.ResolvedPath)
	assert.Equal(t, "/Workspace/root/files", acl.Permissions.Path)
	assert.True(t, acl.CanWrite(user("me@example.com")))
}
