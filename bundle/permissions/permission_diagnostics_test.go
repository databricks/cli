package permissions_test

import (
	"sync/atomic"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/iamutil"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPermissionDiagnosticsApplySuccess(t *testing.T) {
	b := mockBundle(t, []resources.Permission{
		{Level: "CAN_MANAGE", UserName: "testuser@databricks.com"},
	})

	diags := bundle.Apply(t.Context(), b, permissions.PermissionDiagnostics())
	require.NoError(t, diags.Error())
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

// groupCalls counts the group requests the workspace serves.
type groupCalls struct {
	list atomic.Int32 // GET /api/2.0/identity/groups
	get  atomic.Int32 // GET /api/2.0/identity/groups/{group_id}
}

// mockBundle returns a bundle whose workspace has one group, testgroup, that the
// current user belongs to.
func mockBundle(t *testing.T, permissions []resources.Permission) *bundle.Bundle {
	b, _ := mockBundleWithGroups(t, permissions, []string{"testgroup"}, nil)
	return b
}

// mockBundleWithGroups returns a bundle whose workspace creates the groups in member,
// which the current user belongs to, and the groups in nonMember, which it does not.
// The bundle's current user carries the groups resolved from the membership, as the
// populate mutator would set them. The returned counts cover only the group requests
// the bundle makes after this call returns.
func mockBundleWithGroups(t *testing.T, permissions []resources.Permission, member, nonMember []string) (*bundle.Bundle, *groupCalls) {
	server := testserver.New(t)
	calls := &groupCalls{}
	// Handlers registered before AddDefaultHandlers take precedence, so they count and
	// then delegate to the fake.
	server.Handle("GET", "/api/2.0/identity/groups", func(req testserver.Request) any {
		calls.list.Add(1)
		return req.Workspace.IdentityGroups(0, req.URL.Query().Get("page_token"))
	})
	server.Handle("GET", "/api/2.0/identity/groups/{group_id}", func(req testserver.Request) any {
		calls.get.Add(1)
		return req.Workspace.IdentityGroup(req.Vars["group_id"])
	})
	testserver.AddDefaultHandlers(server)
	server.AddGroups("test-token", member, nonMember)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "test-token"})
	require.NoError(t, err)
	groups, err := iamutil.CurrentUserGroups(t.Context(), w)
	require.NoError(t, err)
	calls.list.Store(0)
	calls.get.Store(0)

	b := &bundle.Bundle{
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
	b.SetWorkpaceClient(w)
	return b, calls
}

// A group permission the current user cannot match, with no parent groups in the
// workspace, is reported as not manageable and names the group as a contact.
func TestPermissionDiagnosticsNoGroupMembership(t *testing.T) {
	b, _ := mockBundleWithGroups(t, []resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
	}, nil, nil)

	diags := permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "may include: testgroup.")
}

// A group that exists in the workspace but that the current user is not a member of
// does not grant CAN_MANAGE, and is still named as a contact.
func TestPermissionDiagnosticsGroupExistsNotMember(t *testing.T) {
	b, _ := mockBundleWithGroups(t, []resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "otherteam"},
	}, nil, []string{"otherteam"})

	diags := permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "may include: otherteam.")
}

// A configured group the workspace does not have is named as a contact, not matched.
func TestPermissionDiagnosticsGroupDoesNotExist(t *testing.T) {
	b, _ := mockBundleWithGroups(t, []resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "nosuchgroup"},
	}, nil, nil)

	diags := permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "may include: nosuchgroup.")
}

// A group the current user belongs to, matched by ID, grants CAN_MANAGE. The report
// is then a stale permission set, not a missing grant.
func TestPermissionDiagnosticsGroupMember(t *testing.T) {
	b, _ := mockBundleWithGroups(t, []resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
	}, []string{"testgroup"}, nil)

	diags := permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
	require.Equal(t, diag.CannotChangePathPermissions, diags[0].ID)
}

// Matching a configured group is a comparison with the groups already on the current
// user, so a report makes no group request, whatever the number of configured groups.
func TestPermissionDiagnosticsMembershipMakesNoGroupRequests(t *testing.T) {
	b, calls := mockBundleWithGroups(t, []resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
		{Level: "CAN_MANAGE", GroupName: "otherteam"},
		{Level: "CAN_MANAGE", GroupName: "nosuchgroup"},
	}, []string{"testgroup", "joined"}, []string{"otherteam"})

	permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
	assert.Equal(t, int32(0), calls.list.Load())
	assert.Equal(t, int32(0), calls.get.Load())
}

// A user whose groups were not read, for example because the groups list failed, has
// no group matches, so the configured group is named as a contact.
func TestPermissionDiagnosticsNoGroupsMatchesNothing(t *testing.T) {
	b, _ := mockBundleWithGroups(t, []resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
	}, nil, nil)
	b.Config.Workspace.CurrentUser.Groups = nil

	diags := permissions.ReportPossiblePermissionDenied(t.Context(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "may include: testgroup.")
}
