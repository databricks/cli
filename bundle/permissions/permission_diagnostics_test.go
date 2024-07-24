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
	require.Contains(t, diags[0].Summary, "testuser@databricks.com")
}

func TestPermissionDiagnosticsApplySuccessWithOwner(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "IS_OWNER", UserName: "testuser@databricks.com"},
	})

	diags := permissions.PermissionDiagnostics().Apply(context.Background(), b)
	require.Empty(t, diags)
}

func TestPermissionDiagnosticsPermissionDeniedWithPermission(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
	})

	diags := permissions.ReportPermissionDenied(context.Background(), b, "testpath")
	require.ErrorContains(t, diags.Error(), string(diag.CannotChangePathPermissions))
}

func TestPermissionDiagnosticsPermissionDeniedWithoutPermission(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_VIEW", UserName: "testuser@databricks.com"},
	})

	diags := permissions.ReportPermissionDenied(context.Background(), b, "testpath")
	require.ErrorContains(t, diags.Error(), string(diag.PathPermissionDenied))
}

func TestPermissionDiagnosticsPermissionDeniedNilPermission(t *testing.T) {
	b := mockBundle(nil)

	diags := permissions.ReportPermissionDenied(context.Background(), b, "testpath")
	require.ErrorContains(t, diags.Error(), string(diag.PathPermissionDenied))
}

func TestPermissionDiagnosticsFindOtherOwners(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})

	diags := permissions.ReportPermissionDenied(context.Background(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "include: alice@databricks.com")
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
