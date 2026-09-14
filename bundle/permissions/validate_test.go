package permissions

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func applyValidate(t *testing.T, b *bundle.Bundle) diag.Diagnostics {
	t.Helper()
	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	return bundle.Apply(t.Context(), b, ValidateWorkspaceSharedPermissions())
}

func TestValidateWorkspaceSharedPermissions_RootShared_NoWarningWithUsersManage(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Workspace/Shared/foo/bar",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, GroupName: "users"},
			},
		},
	}
	diags := applyValidate(t, b)
	require.Empty(t, diags)
}

func TestValidateWorkspaceSharedPermissions_StateShared_WarnWithoutUsersManage(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:  "/Workspace/Users/user@example.test",
				StatePath: "/Workspace/Shared/state",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "user@example.test"},
			},
		},
	}
	diags := applyValidate(t, b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Warning, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "state path")
	assert.Contains(t, diags[0].Summary, "/Workspace/Shared/state")
}

func TestValidateWorkspaceSharedPermissions_StateShared_NoWarningWithUsersManage(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:  "/Workspace/Users/user@example.test",
				StatePath: "/Workspace/Shared/state",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, GroupName: "users"},
			},
		},
	}
	diags := applyValidate(t, b)
	require.Empty(t, diags)
}

func TestValidateWorkspaceSharedPermissions_NoSharedPaths_NoWarning(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath:  "/Workspace/Users/user@example.test/bundle",
				StatePath: "/Workspace/Users/user@example.test/other-state",
			},
		},
	}
	diags := applyValidate(t, b)
	require.Empty(t, diags)
}

func TestStatePathUnderRootPath(t *testing.T) {
	cases := []struct {
		name      string
		rootPath  string
		statePath string
		want      bool
	}{
		{name: "default nested state", rootPath: "/Workspace/Users/me/bundle", statePath: "/Workspace/Users/me/bundle/state", want: true},
		{name: "equal paths", rootPath: "/Workspace/Users/me/bundle", statePath: "/Workspace/Users/me/bundle", want: true},
		{name: "separate folder", rootPath: "/Workspace/Users/me/bundle", statePath: "/Workspace/Shared/state", want: false},
		{name: "sibling prefix is not nested", rootPath: "/Workspace/Users/me/bundle", statePath: "/Workspace/Users/me/bundle-2/state", want: false},
		{name: "empty state defaults to nested", rootPath: "/Workspace/Users/me/bundle", statePath: "", want: true},
		{name: "empty root defaults to nested", rootPath: "", statePath: "/Workspace/Shared/state", want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, statePathUnderRootPath(tc.rootPath, tc.statePath))
		})
	}
}
