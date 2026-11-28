package permissions_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/require"
)

func TestPermissionDiagnosticsApplySuccess(t *testing.T) {
	b := mockBundle(t, []resources.Permission{
		{Level: "CAN_MANAGE", UserName: "testuser@databricks.com"},
	})

	diags := bundle.Apply(t.Context(), b, permissions.PermissionDiagnostics())
	require.NoError(t, diags.Error())
}

// When the groups could not be read, a grant through a group may already apply, so the
// recommendation to add the identity is withheld.
func TestPermissionDiagnosticsSkipsRecommendationWhenGroupsUnknown(t *testing.T) {
	b := mockBundle(t, []resources.Permission{
		{Level: "CAN_VIEW", UserName: "testuser@databricks.com"},
	})
	require.NotEmpty(t, bundle.Apply(t.Context(), b, permissions.PermissionDiagnostics()))

	b.CurrentUserGroupsUnknown = true
	require.Empty(t, bundle.Apply(t.Context(), b, permissions.PermissionDiagnostics()))
}

func TestPermissionDiagnosticsEmpty(t *testing.T) {
	b := mockBundle(t, nil)

	diags := bundle.Apply(t.Context(), b, permissions.PermissionDiagnostics())
	require.NoError(t, diags.Error())
}

func TestPermissionDiagnosticsApplyFail(t *testing.T) {
	b := mockBundle(t, []resources.Permission{
		{Level: "CAN_VIEW", UserName: "testuser@databricks.com"},
	})

	diags := bundle.Apply(t.Context(), b, permissions.PermissionDiagnostics())
	require.Equal(t, diag.Recommendation, diags[0].Severity)

	expectedMsg := "permissions section should explicitly include the current deployment identity " +
		"'testuser@databricks.com' or one of its groups\n" +
		"If it is not included, CAN_MANAGE permissions are only applied if the present identity is used to deploy.\n\n" +
		"Consider using a adding a top-level permissions section such as the following:\n\n" +
		"  permissions:\n" +
		"    - user_name: testuser@databricks.com\n" +
		"      level: CAN_MANAGE\n\n" +
		"See https://docs.databricks.com/dev-tools/bundles/permissions.html to learn more about permission configuration."

	require.Contains(t, diags[0].Summary, expectedMsg)
}

// mockBundle returns a bundle whose current user belongs to testgroup.
func mockBundle(t *testing.T, permissions []resources.Permission) *bundle.Bundle {
	return mockBundleWithGroups(t, permissions, "testgroup")
}

// mockBundleWithGroups returns a bundle whose current user carries the groups named by
// memberOf, as the populate mutator sets them. Permission matching compares names only, so
// the bundle needs no workspace.
func mockBundleWithGroups(t *testing.T, permissions []resources.Permission, memberOf ...string) *bundle.Bundle {
	t.Helper()
	var groups []iam.ComplexValue
	for _, name := range memberOf {
		groups = append(groups, iam.ComplexValue{Value: "id-" + name, Display: name})
	}
	return &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{
						UserName:    "testuser@databricks.com",
						DisplayName: "Test User",
						Groups:      groups,
					},
				},
			},
			Permissions: permissions,
		},
	}
}

// A group permission the current user cannot match is reported as not manageable and names
// the group as a contact.
func TestPermissionDiagnosticsNoGroupMembership(t *testing.T) {
	b := mockBundleWithGroups(t, []resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
	})

	diags := permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "may include: testgroup.")
}

// A configured group the current user is not a member of does not grant CAN_MANAGE, and is
// still named as a contact.
func TestPermissionDiagnosticsGroupNotMember(t *testing.T) {
	b := mockBundleWithGroups(t, []resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "otherteam"},
	})

	diags := permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "may include: otherteam.")
}

// A group the current user belongs to, matched by name, grants CAN_MANAGE. The report is
// then a stale permission set, not a missing grant.
func TestPermissionDiagnosticsGroupMember(t *testing.T) {
	b := mockBundleWithGroups(t, []resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
	}, "testgroup")

	diags := permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
	require.Equal(t, diag.CannotChangePathPermissions, diags[0].ID)
}

// Workspace-local groups are matched by name like account groups: a CAN_MANAGE grant to
// admins or users that the current user belongs to counts as manageable.
func TestPermissionDiagnosticsWorkspaceLocalGroupGrantsCanManage(t *testing.T) {
	for _, name := range []string{"admins", "users"} {
		t.Run(name, func(t *testing.T) {
			b := mockBundleWithGroups(t, []resources.Permission{
				{Level: "CAN_MANAGE", GroupName: name},
			}, name)

			require.Empty(t, bundle.Apply(t.Context(), b, permissions.PermissionDiagnostics()))
			diags := permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
			require.Equal(t, diag.CannotChangePathPermissions, diags[0].ID)
		})
	}
}

// A user whose groups were not read has no group matches, so the configured group is named
// as a contact.
func TestPermissionDiagnosticsNoGroupsMatchesNothing(t *testing.T) {
	b := mockBundleWithGroups(t, []resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
	})
	b.Config.Workspace.CurrentUser.Groups = nil

	diags := permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "may include: testgroup.")
}
