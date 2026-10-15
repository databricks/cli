package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/sdkclients"
	"github.com/databricks/cli/libs/testserver"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/require"
)

// folderPermissionsClients returns clients for a workspace in which the paths
// in existing are directories with the given object IDs and every other path
// does not exist. Directory 1234 has the given access control list.
func folderPermissionsClients(t *testing.T, existing map[string]int64, acl []map[string]any) *sdkclients.Clients {
	server := testserver.New(t)
	server.Handle("GET", "/api/2.0/workspace/get-status", func(req testserver.Request) any {
		path := req.URL.Query().Get("path")
		id, ok := existing[path]
		if !ok {
			return testserver.Response{
				StatusCode: 404,
				Body:       map[string]string{"error_code": "RESOURCE_DOES_NOT_EXIST", "message": path + " does not exist"},
			}
		}
		return map[string]any{"object_id": id, "object_type": "DIRECTORY", "path": path}
	})
	server.Handle("GET", "/api/2.0/permissions/directories/1234", func(testserver.Request) any {
		return map[string]any{"object_id": "1234", "access_control_list": acl}
	})
	w, err := sdkclients.New(t.Context(), &sdkconfig.Config{
		Host:    server.URL,
		Token:   "testtoken",
		Loaders: []sdkconfig.Loader{sdkconfig.ConfigAttributes},
	})
	require.NoError(t, err)
	return w
}

func manage(userName string) map[string]any {
	return map[string]any{
		"user_name":       userName,
		"all_permissions": []map[string]any{{"permission_level": "CAN_MANAGE"}},
	}
}

func TestFolderPermissionsInheritedWhenRootPathDoesNotExist(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Workspace/Users/foo@bar.test",
				ArtifactPath: "/Workspace/Users/otherfoo@bar.test/artifacts",
				FilePath:     "/Workspace/Users/foo@bar.test/files",
				StatePath:    "/Workspace/Users/foo@bar.test/state",
				ResourcePath: "/Workspace/Users/foo@bar.test/resources",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "foo@bar.test"},
			},
		},
	}

	b.SetWorkpaceClient(folderPermissionsClients(t, map[string]int64{"/Workspace": 1234}, []map[string]any{
		manage("foo@bar.test"),
	}))
	diags := ValidateFolderPermissions().Apply(t.Context(), b)
	require.Empty(t, diags)
}

func TestValidateFolderPermissionsFailsOnMissingBundlePermission(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Workspace/Users/foo@bar.test",
				ArtifactPath: "/Workspace/Users/foo@bar.test/artifacts",
				FilePath:     "/Workspace/Users/foo@bar.test/files",
				StatePath:    "/Workspace/Users/foo@bar.test/state",
				ResourcePath: "/Workspace/Users/foo@bar.test/resources",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "foo@bar.test"},
			},
		},
	}

	b.SetWorkpaceClient(folderPermissionsClients(t, map[string]int64{"/Workspace/Users/foo@bar.test": 1234}, []map[string]any{
		manage("foo@bar.test"),
		manage("foo2@bar.test"),
	}))
	diags := ValidateFolderPermissions().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	require.Equal(t, "workspace folder has permissions not configured in bundle", diags[0].Summary)
	require.Equal(t, diag.Warning, diags[0].Severity)
	expectedDetail := "The following permissions apply to the workspace folder at \"/Workspace/Users/foo@bar.test\" " +
		"but are not configured in the bundle:\n- level: CAN_MANAGE, user_name: foo2@bar.test\n\n" +
		"Add them to your bundle permissions or remove them from the folder.\n" +
		"See https://docs.databricks.com/dev-tools/bundles/permissions"
	require.Equal(t, expectedDetail, diags[0].Detail)
}

func TestValidateFolderPermissionsFailsOnPermissionMismatch(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:     "/Workspace/Users/foo@bar.test",
				ArtifactPath: "/Workspace/Users/foo@bar.test/artifacts",
				FilePath:     "/Workspace/Users/foo@bar.test/files",
				StatePath:    "/Workspace/Users/foo@bar.test/state",
				ResourcePath: "/Workspace/Users/foo@bar.test/resources",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "foo@bar.test"},
			},
		},
	}

	b.SetWorkpaceClient(folderPermissionsClients(t, map[string]int64{"/Workspace/Users/foo@bar.test": 1234}, []map[string]any{
		manage("foo2@bar.test"),
	}))
	diags := ValidateFolderPermissions().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	require.Equal(t, "workspace folder has permissions not configured in bundle", diags[0].Summary)
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
				{Level: permissions.CAN_MANAGE, UserName: "foo@bar.test"},
			},
		},
	}

	b.SetWorkpaceClient(folderPermissionsClients(t, nil, nil))
	diags := ValidateFolderPermissions().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	require.Equal(t, "folder / and its parent folders do not exist", diags[0].Summary)
	require.Equal(t, diag.Error, diags[0].Severity)
}
