package permissions

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestApplySuccess(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "testuser@databricks.com"},
	})

	diags := PermissionDiagnostics().Apply(context.Background(), b)
	require.NoError(t, diags.Error())
}

func TestApplyFail(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_VIEW", UserName: "testuser@databricks.com"},
	})

	diags := PermissionDiagnostics().Apply(context.Background(), b)
	require.Equal(t, diags[0].Severity, diag.Warning)
	require.Contains(t, diags[0].Summary, "testuser@databricks.com")
}

func TestApplySuccesWithOwner(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "IS_OWNER", UserName: "testuser@databricks.com"},
	})

	diags := PermissionDiagnostics().Apply(context.Background(), b)
	require.Equal(t, len(diags), 0)
}

func TestPermissionDeniedWithPermission(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
	})

	diags := ReportPermissionDenied(context.Background(), b, "testpath")
	require.ErrorContains(t, diags.Error(), string(diag.CannotChangePathPermissions))
}

func TestPermissionDeniedWithoutPermission(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_VIEW", UserName: "testuser@databricks.com"},
	})

	diags := ReportPermissionDenied(context.Background(), b, "testpath")
	require.ErrorContains(t, diags.Error(), string(diag.PathPermissionDenied))
}

func TestPermissionDeniedNilPermission(t *testing.T) {
	b := mockBundle(nil)

	diags := ReportPermissionDenied(context.Background(), b, "testpath")
	require.ErrorContains(t, diags.Error(), string(diag.PathPermissionDenied))
}

func TestFindOtherOwners(t *testing.T) {
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "testgroup"},
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})

	diags := ReportPermissionDenied(context.Background(), b, "testpath")
	require.ErrorContains(t, diags.Error(), "include: alice@databricks.com")
}

func TestReportTerraformError1(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})
	err := TryReportTerraformPermissionError(ctx, b, errors.New(`Error: terraform apply: exit status 1

Error: cannot update permissions: ...

	with databricks_pipeline.my_project_pipeline,
	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:
	39:       }`)).Error()
	require.ErrorContains(t, err, string(diag.ResourcePermissionDenied))
}

func TestReportTerraformError2(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})
	err := TryReportTerraformPermissionError(ctx, b, errors.New(`Error: terraform apply: exit status 1

Error: cannot read pipeline: User xyz does not have View permissions on pipeline 4521dbb6-42aa-418c-b94d-b5f4859a3454.

	with databricks_pipeline.my_project_pipeline,
	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:
	39:       }`)).Error()
	require.ErrorContains(t, err, string(diag.ResourcePermissionDenied))
}

func TestReportTerraformError3(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})
	err := TryReportTerraformPermissionError(ctx, b, errors.New(`Error: terraform apply: exit status 1

	Error: cannot read permissions: 1706906c-c0a2-4c25-9f57-3a7aa3cb8b90 does not have Owner permissions on Job with ID: ElasticJobId(28263044278868). Please contact the owner or an administrator for access.

	with databricks_pipeline.my_project_pipeline,
	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:
	39:       }`)).Error()
	require.ErrorContains(t, err, string(diag.ResourcePermissionDenied))
}

func TestReportTerraformErrorNotOwner(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})
	b.Config.RunAs = &jobs.JobRunAs{
		UserName: "testuser@databricks.com",
	}
	err := TryReportTerraformPermissionError(ctx, b, errors.New(`Error: terraform apply: exit status 1

Error: cannot read pipeline: User xyz does not have View permissions on pipeline 4521dbb6-42aa-418c-b94d-b5f4859a3454.

	with databricks_pipeline.my_project_pipeline,
	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:
	39:       }`)).Error()
	require.ErrorContains(t, err, string(diag.ResourcePermissionDenied))
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
