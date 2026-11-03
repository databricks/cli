package permissions_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestPermissionDiagnosticsApplySuccess(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "testuser@databricks.com"},
	})

	diags := bundle.Apply(t.Context(), b, permissions.PermissionDiagnostics())
	require.NoError(t, diags.Error())
}

func TestPermissionDiagnosticsEmpty(t *testing.T) {
	b := mockBundle(nil)

	diags := bundle.Apply(t.Context(), b, permissions.PermissionDiagnostics())
	require.NoError(t, diags.Error())
}

func TestPermissionDiagnosticsApplyFail(t *testing.T) {
	b := mockBundle([]resources.Permission{
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

func TestPermissionDiagnosticsRunAsFileAccess(t *testing.T) {
	deployerPermission := resources.Permission{Level: "CAN_MANAGE", UserName: "testuser@databricks.com"}
	servicePrincipalName := "aaaaaaaa-bbbb-4ccc-dddd-eeeeeeeeeeee"
	servicePrincipal := jobs.JobRunAs{ServicePrincipalName: servicePrincipalName}

	testCases := []struct {
		name           string
		runAs          jobs.JobRunAs
		permissions    []resources.Permission
		currentUser    string
		syncMode       config.SyncMode
		filePath       string
		wantSummary    string
		wantSuggestion string
	}{
		{
			name:           "service principal without permissions",
			runAs:          servicePrincipal,
			wantSummary:    "is not in top-level permissions for bundle files",
			wantSuggestion: "service_principal_name: " + servicePrincipalName + "\n      level: CAN_VIEW",
		},
		{
			name:           "user without permissions",
			runAs:          jobs.JobRunAs{UserName: "runner@example.test"},
			wantSummary:    "is not in top-level permissions for bundle files",
			wantSuggestion: "user_name: runner@example.test\n      level: CAN_VIEW",
		},
		{
			name:           "group without permissions",
			runAs:          jobs.JobRunAs{GroupName: "runners"},
			wantSummary:    "is not in top-level permissions for bundle files",
			wantSuggestion: "group_name: runners\n      level: CAN_VIEW",
		},
		{
			name:           "unrelated group permission",
			runAs:          servicePrincipal,
			permissions:    []resources.Permission{deployerPermission, {Level: "CAN_VIEW", GroupName: "runners"}},
			wantSummary:    "is not in top-level permissions for bundle files",
			wantSuggestion: "service_principal_name: " + servicePrincipalName,
		},
		{
			name:        "service principal with view permission",
			runAs:       servicePrincipal,
			permissions: []resources.Permission{deployerPermission, {Level: "CAN_VIEW", ServicePrincipalName: servicePrincipalName}},
		},
		{
			name:        "service principal with run permission",
			runAs:       servicePrincipal,
			permissions: []resources.Permission{deployerPermission, {Level: "CAN_RUN", ServicePrincipalName: servicePrincipalName}},
		},
		{
			name:        "service principal with manage permission",
			runAs:       servicePrincipal,
			permissions: []resources.Permission{deployerPermission, {Level: "CAN_MANAGE", ServicePrincipalName: servicePrincipalName}},
		},
		{
			name:        "group with view permission",
			runAs:       jobs.JobRunAs{GroupName: "runners"},
			permissions: []resources.Permission{deployerPermission, {Level: "CAN_VIEW", GroupName: "runners"}},
		},
		{
			name:  "run as deployment user",
			runAs: jobs.JobRunAs{UserName: "testuser@databricks.com"},
		},
		{
			name:        "run as deployment service principal",
			runAs:       servicePrincipal,
			currentUser: servicePrincipal.ServicePrincipalName,
		},
		{
			name:           "source-linked files",
			runAs:          servicePrincipal,
			syncMode:       config.SyncModeInPlace,
			wantSummary:    "may not be able to read source-linked files",
			wantSuggestion: "CAN_READ access to that folder; top-level bundle permissions do not manage source-linked files",
		},
		{
			name:           "source-linked files with top-level permissions",
			runAs:          servicePrincipal,
			permissions:    []resources.Permission{deployerPermission, {Level: "CAN_VIEW", ServicePrincipalName: servicePrincipalName}},
			syncMode:       config.SyncModeInPlace,
			wantSummary:    "may not be able to read source-linked files",
			wantSuggestion: "CAN_READ access to that folder",
		},
		{
			name:           "immutable bundle without permissions",
			runAs:          servicePrincipal,
			syncMode:       config.SyncModeImmutable,
			wantSummary:    "is not in top-level permissions for bundle files",
			wantSuggestion: "level: CAN_VIEW",
		},
		{
			name:           "files under Shared",
			runAs:          servicePrincipal,
			filePath:       "/Workspace/Shared/team/files",
			wantSummary:    "may not be able to read files in /Workspace/Shared",
			wantSuggestion: "Check that the run_as identity has CAN_READ access to /Workspace/Shared/team/files. Top-level bundle permissions do not manage files in /Workspace/Shared.",
		},
		{
			name:           "files under Shared with top-level permissions",
			runAs:          servicePrincipal,
			permissions:    []resources.Permission{deployerPermission, {Level: "CAN_VIEW", ServicePrincipalName: servicePrincipalName}},
			filePath:       "/Workspace/Shared/team/files",
			wantSummary:    "may not be able to read files in /Workspace/Shared",
			wantSuggestion: "Top-level bundle permissions do not manage files in /Workspace/Shared.",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			deployment := mockBundle(testCase.permissions)
			deployment.Config.RunAs = &testCase.runAs
			deployment.Config.Workspace.SyncMode = testCase.syncMode
			deployment.Config.Workspace.FilePath = testCase.filePath
			if testCase.currentUser != "" {
				deployment.Config.Workspace.CurrentUser.User.UserName = testCase.currentUser
			}

			diags := bundle.Apply(t.Context(), deployment, permissions.PermissionDiagnostics())
			if testCase.wantSummary == "" {
				require.Empty(t, diags)
				return
			}

			require.Len(t, diags, 1)
			require.Equal(t, diag.Warning, diags[0].Severity)
			require.Contains(t, diags[0].Summary, testCase.wantSummary)
			require.Contains(t, diags[0].Detail, testCase.wantSuggestion)
			require.NoError(t, diags.Error())
		})
	}
}

func TestPermissionDiagnosticsResourcePermissionDoesNotCoverRunAsFiles(t *testing.T) {
	servicePrincipalName := "aaaaaaaa-bbbb-4ccc-dddd-eeeeeeeeeeee"
	deployment := mockBundle(nil)
	deployment.Config.RunAs = &jobs.JobRunAs{ServicePrincipalName: servicePrincipalName}
	deployment.Config.Resources.Jobs = map[string]*resources.Job{
		"job": {
			Permissions: []resources.JobPermission{{Level: "CAN_VIEW", ServicePrincipalName: servicePrincipalName}},
		},
	}

	diags := bundle.Apply(t.Context(), deployment, permissions.PermissionDiagnostics())
	require.Len(t, diags, 1)
	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Contains(t, diags[0].Detail, "level: CAN_VIEW")
}

func TestPermissionDiagnosticsRunAsAndDeployerWarnings(t *testing.T) {
	deployment := mockBundle([]resources.Permission{{Level: "CAN_VIEW", GroupName: "other_group"}})
	deployment.Config.RunAs = &jobs.JobRunAs{UserName: "runner@example.test"}

	diags := bundle.Apply(t.Context(), deployment, permissions.PermissionDiagnostics())
	require.Len(t, diags, 2)
	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Equal(t, diag.Recommendation, diags[1].Severity)
}

func mockBundle(permissions []resources.Permission) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{
						UserName:    "testuser@databricks.com",
						DisplayName: "Test User",
						Groups: []iam.ComplexValue{
							{Display: "testgroup"},
						},
					},
				},
			},
			Permissions: permissions,
		},
	}
}
