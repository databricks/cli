package permissions

import (
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

func TestWorkspacePathPermissionsCompare(t *testing.T) {
	testCases := []struct {
		perms    []resources.Permission
		acl      []workspace.WorkspaceObjectAccessControlResponse
		expected diag.Diagnostics
	}{
		{
			perms: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "foo@bar.com"},
			},
			acl: []workspace.WorkspaceObjectAccessControlResponse{
				{
					UserName: "foo@bar.com",
					AllPermissions: []workspace.WorkspaceObjectPermission{
						{PermissionLevel: "CAN_MANAGE"},
					},
				},
			},
			expected: nil,
		},
		{
			perms: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "foo@bar.com"},
			},
			acl: []workspace.WorkspaceObjectAccessControlResponse{
				{
					UserName: "foo@bar.com",
					AllPermissions: []workspace.WorkspaceObjectPermission{
						{PermissionLevel: "CAN_MANAGE"},
					},
				},
				{
					GroupName: "admins",
					AllPermissions: []workspace.WorkspaceObjectPermission{
						{PermissionLevel: "CAN_MANAGE"},
					},
				},
			},
			expected: nil,
		},
		{
			perms: []resources.Permission{
				{Level: CAN_VIEW, UserName: "foo@bar.com"},
				{Level: CAN_MANAGE, ServicePrincipalName: "sp.com"},
			},
			acl: []workspace.WorkspaceObjectAccessControlResponse{
				{
					UserName: "foo@bar.com",
					AllPermissions: []workspace.WorkspaceObjectPermission{
						{PermissionLevel: "CAN_READ"},
					},
				},
			},
			expected: nil,
		},
		{
			perms: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "foo@bar.com"},
			},
			acl: []workspace.WorkspaceObjectAccessControlResponse{
				{
					UserName: "foo@bar.com",
					AllPermissions: []workspace.WorkspaceObjectPermission{
						{PermissionLevel: "CAN_MANAGE"},
					},
				},
				{
					GroupName: "foo",
					AllPermissions: []workspace.WorkspaceObjectPermission{
						{PermissionLevel: "CAN_MANAGE"},
					},
				},
			},
			expected: diag.Diagnostics{
				{
					Severity: diag.Warning,
					Summary:  "workspace folder has permissions not configured in bundle",
					Detail: "The following permissions apply to the workspace folder at \"path\" " +
						"but are not configured in the bundle:\n- level: CAN_MANAGE, group_name: foo\n\n" +
						"Add them to your bundle permissions or remove them from the folder.\n" +
						"See https://docs.databricks.com/dev-tools/bundles/permissions",
				},
			},
		},
		{
			perms: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "foo@bar.com"},
			},
			acl: []workspace.WorkspaceObjectAccessControlResponse{
				{
					UserName: "foo2@bar.com",
					AllPermissions: []workspace.WorkspaceObjectPermission{
						{PermissionLevel: "CAN_MANAGE"},
					},
				},
			},
			expected: diag.Diagnostics{
				{
					Severity: diag.Warning,
					Summary:  "workspace folder has permissions not configured in bundle",
					Detail: "The following permissions apply to the workspace folder at \"path\" " +
						"but are not configured in the bundle:\n- level: CAN_MANAGE, user_name: foo2@bar.com\n\n" +
						"Add them to your bundle permissions or remove them from the folder.\n" +
						"See https://docs.databricks.com/dev-tools/bundles/permissions",
				},
			},
		},
	}

	for _, tc := range testCases {
		wp := ObjectAclToResourcePermissions("path", tc.acl)
		diags := wp.Compare(tc.perms)
		require.Equal(t, tc.expected, diags)
	}
}
