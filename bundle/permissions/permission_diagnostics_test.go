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

	diags := bundle.Apply(context.Background(), b, permissions.PermissionDiagnostics())
	require.NoError(t, diags.Error())
}

func TestPermissionDiagnosticsEmpty(t *testing.T) {
	b := mockBundle(nil)

	diags := bundle.Apply(context.Background(), b, permissions.PermissionDiagnostics())
	require.NoError(t, diags.Error())
}

func TestPermissionDiagnosticsApplyFail(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_VIEW", UserName: "testuser@databricks.com"},
	})

	diags := bundle.Apply(context.Background(), b, permissions.PermissionDiagnostics())
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
