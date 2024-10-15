package permissions_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/stretchr/testify/require"
)

func TestPermissionsReportPermissionDeniedWithGroup(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
	})

	diags := permissions.ReportPossiblePermissionDenied(context.Background(), b, "testpath")
	expected := "EPERM3: unable to deploy to testpath as testuser@databricks.com. Cannot apply local deployment permissions.\n" +
		"For assistance, contact the owners of this project.\n" +
		"They can redeploy the project to apply the latest set of permissions.\n" +
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions."
	require.ErrorContains(t, diags.Error(), expected)
}

func TestPermissionsReportPermissionDeniedWithOtherGroup(t *testing.T) {
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

func TestPermissionsReportPermissionDeniedWithoutPermission(t *testing.T) {
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

func TestPermissionsReportPermissionDeniedNilPermission(t *testing.T) {
	b := mockBundle(nil)

	diags := permissions.ReportPossiblePermissionDenied(context.Background(), b, "testpath")
	expected := "EPERM1: unable to deploy to testpath as testuser@databricks.com.\n" +
		"Please make sure the current user or one of their groups is listed under the permissions of this bundle.\n" +
		"For assistance, contact the owners of this project.\n" +
		"They may need to redeploy the bundle to apply the new permissions.\n" +
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions"
	require.ErrorContains(t, diags.Error(), expected)
}

func TestPermissionsReportFindOtherOwners(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})

	diags := permissions.ReportPossiblePermissionDenied(context.Background(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "EPERM3: unable to deploy to testpath as testuser@databricks.com. Cannot apply local deployment permissions.\n"+
		"For assistance, users or groups with appropriate permissions may include: alice@databricks.com.\n"+
		"They can redeploy the project to apply the latest set of permissions.\n"+
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions.")
}
