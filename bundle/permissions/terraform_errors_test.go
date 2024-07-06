package permissions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestTryExtendTerraformPermissionError1(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})
	err := permissions.TryExtendTerraformPermissionError(ctx, b, errors.New(`Error: terraform apply: exit status 1

Error: cannot update permissions: ...

	with databricks_pipeline.my_project_pipeline,
	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:
	39:       }`)).Error()
	require.ErrorContains(t, err, string(diag.ResourcePermissionDenied))
}

func TestTryExtendTerraformPermissionError2(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})
	err := permissions.TryExtendTerraformPermissionError(ctx, b, errors.New(`Error: terraform apply: exit status 1

Error: cannot read pipeline: User xyz does not have View permissions on pipeline 4521dbb6-42aa-418c-b94d-b5f4859a3454.

	with databricks_pipeline.my_project_pipeline,
	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:
	39:       }`)).Error()
	require.ErrorContains(t, err, string(diag.ResourcePermissionDenied))
}

func TestTryExtendTerraformPermissionError3(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})
	err := permissions.TryExtendTerraformPermissionError(ctx, b, errors.New(`Error: terraform apply: exit status 1

	Error: cannot read permissions: 1706906c-c0a2-4c25-9f57-3a7aa3cb8b90 does not have Owner permissions on Job with ID: ElasticJobId(28263044278868). Please contact the owner or an administrator for access.

	with databricks_pipeline.my_project_pipeline,
	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:
	39:       }`)).Error()
	require.ErrorContains(t, err, string(diag.ResourcePermissionDenied))
}

func TestTryExtendTerraformPermissionErrorNotOwner(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})
	b.Config.RunAs = &jobs.JobRunAs{
		UserName: "testuser@databricks.com",
	}
	err := permissions.TryExtendTerraformPermissionError(ctx, b, errors.New(`Error: terraform apply: exit status 1

Error: cannot read pipeline: User xyz does not have View permissions on pipeline 4521dbb6-42aa-418c-b94d-b5f4859a3454.

	with databricks_pipeline.my_project_pipeline,
	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:
	39:       }`)).Error()
	require.ErrorContains(t, err, string(diag.ResourcePermissionDenied))
}
