package permissions_test

import (
	"context"
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
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "testuser@databricks.com"},
	})

	diags := permissions.PermissionDiagnostics().Apply(context.Background(), b)
	require.NoError(t, diags.Error())
}

func TestPermissionDiagnosticsApplyFail(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_VIEW", UserName: "testuser@databricks.com"},
	})

	diags := permissions.PermissionDiagnostics().Apply(context.Background(), b)
	require.Equal(t, diags[0].Severity, diag.Warning)
	require.Contains(t, diags[0].Summary, "permissions section should include testuser@databricks.com or one of their groups with CAN_MANAGE permissions")
}

func TestPermissionDiagnosticsApplySuccessWithOwner(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "IS_OWNER", UserName: "testuser@databricks.com"},
	})

	diags := permissions.PermissionDiagnostics().Apply(context.Background(), b)
	require.Empty(t, diags)
}

func TestPermissionDiagnosticsPermissionDeniedWithGroup(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
	})

	diags := permissions.ReportPossiblePermissionDenied(context.Background(), b, "testpath")
	expected := "EPERM1: unable to deploy to testpath as testuser@databricks.com.\n" +
		"Please make sure the current user or one of their groups is listed under the permissions of this bundle.\n" +
		"For assistance, users or groups with appropriate permissions may include: testgroup@databricks.com.\n" +
		"They may need to redeploy the bundle to apply the new permissions.\n" +
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions."
	require.ErrorContains(t, diags.Error(), expected)
}

func TestPermissionDiagnosticsPermissionDeniedWithOtherGroup(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "othergroup"},
	})

	diags := permissions.ReportPossiblePermissionDenied(context.Background(), b, "testpath")
	expected := "EPERM1: unable to deploy to testpath as testuser@databricks.com.\n" +
		"Please make sure the current user or one of their groups is listed under the permissions of this bundle.\n" +
		"For assistance, users or groups with appropriate permissions may include: othergroup.\n" +
		"They may need to redeploy the bundle to apply the new permissions.\n" +
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions."
	require.ErrorContains(t, diags.Error(), expected)
}

func TestPermissionDiagnosticsPermissionDeniedWithoutPermission(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_VIEW", UserName: "testuser@databricks.com"},
	})

	diags := permissions.ReportPossiblePermissionDenied(context.Background(), b, "testpath")
	expected := "EPERM1: unable to deploy to testpath as testuser@databricks.com.\n" +
		"Please make sure the current user or one of their groups is listed under the permissions of this bundle.\n" +
		"For assistance, contact the owners of this project.\n" +
		"They may need to redeploy the bundle to apply the new permissions.\n" +
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions."
	require.ErrorContains(t, diags.Error(), expected)
}

func TestPermissionDiagnosticsPermissionDeniedNilPermission(t *testing.T) {
	b := mockBundle(nil)

	diags := permissions.ReportPossiblePermissionDenied(context.Background(), b, "testpath")
	expected := "EPERM1: unable to deploy to testpath as testuser@databricks.com.\n" +
		"Please make sure the current user or one of their groups is listed under the permissions of this bundle.\n" +
		"For assistance, contact the owners of this project.\n" +
		"They may need to redeploy the bundle to apply the new permissions.\n" +
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions"
	require.ErrorContains(t, diags.Error(), expected)
}

func TestPermissionDiagnosticsFindOtherOwners(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})

	diags := permissions.ReportPossiblePermissionDenied(context.Background(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "EPERM3: access denied while updating deployment permissions as testuser@databricks.com.\n"+
		"For assistance, users or groups with appropriate permissions may include: alice@databricks.com.\n"+
		"They can redeploy the project to apply the latest set of permissions.\n"+
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions.")
}

func mockBundle(permissions []resources.Permission) *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "testuser@databricks.com",
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
