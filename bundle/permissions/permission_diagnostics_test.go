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
	require.Equal(t, diag.Warning, diags[0].Severity)
	require.Contains(t, diags[0].Summary, "permissions section should include testuser@databricks.com or one of their groups with CAN_MANAGE permissions")
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
