package permissions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestTryExtendTerraformPermissionError1(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
	})
	err := permissions.TryExtendTerraformPermissionError(ctx, b, errors.New("Error: terraform apply: exit status 1\n"+
		"\n"+
		"Error: cannot update permissions: ...\n"+
		"\n"+
		"	with databricks_pipeline.my_project_pipeline,\n"+
		"	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:\n"+
		"	39:       }")).Error()

	expected := "EPERM2: permission denied creating or updating my_project_pipeline.\n" +
		"For assistance, users or groups with appropriate permissions may include: alice@databricks.com.\n" +
		"They can redeploy the project to apply the latest set of permissions.\n" +
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions"

	require.ErrorContains(t, err, expected)
}

func TestTryExtendTerraformPermissionError2(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "alice@databricks.com"},
		{Level: "CAN_MANAGE", UserName: "bob@databricks.com"},
	})
	err := permissions.TryExtendTerraformPermissionError(ctx, b, errors.New("Error: terraform apply: exit status 1\n"+
		"\n"+
		"Error: cannot read pipeline: User xyz does not have View permissions on pipeline 4521dbb6-42aa-418c-b94d-b5f4859a3454.\n"+
		"\n"+
		"	with databricks_pipeline.my_project_pipeline,\n"+
		"	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:\n"+
		"	39:       }")).Error()

	expected := "EPERM2: permission denied creating or updating my_project_pipeline.\n" +
		"For assistance, users or groups with appropriate permissions may include: alice@databricks.com, bob@databricks.com.\n" +
		"They can redeploy the project to apply the latest set of permissions.\n" +
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions."
	require.ErrorContains(t, err, expected)
}

func TestTryExtendTerraformPermissionError3(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", UserName: "testuser@databricks.com"},
	})
	err := permissions.TryExtendTerraformPermissionError(ctx, b, errors.New("Error: terraform apply: exit status 1\n"+
		"\n"+
		"Error: cannot read permissions: 1706906c-c0a2-4c25-9f57-3a7aa3cb8b90 does not have Owner permissions on Job with ID: ElasticJobId(28263044278868). Please contact the owner or an administrator for access.\n"+
		"\n"+
		"	with databricks_pipeline.my_project_pipeline,\n"+
		"	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:\n"+
		"	39:       }")).Error()

	expected := "EPERM2: permission denied creating or updating my_project_pipeline.\n" +
		"For assistance, contact the owners of this project.\n" +
		"They can redeploy the project to apply the latest set of permissions.\n" +
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions."
	require.ErrorContains(t, err, expected)
}

func TestTryExtendTerraformPermissionErrorNotOwner(t *testing.T) {
	ctx := context.Background()
	b := mockBundle([]resources.Permission{
		{Level: "CAN_MANAGE", GroupName: "data_team@databricks.com"},
	})
	b.Config.RunAs = &jobs.JobRunAs{
		UserName: "testuser@databricks.com",
	}
	err := permissions.TryExtendTerraformPermissionError(ctx, b, errors.New("Error: terraform apply: exit status 1\n"+
		"\n"+
		"Error: cannot read pipeline: User xyz does not have View permissions on pipeline 4521dbb6-42aa-418c-b94d-b5f4859a3454.\n"+
		"\n"+
		"	with databricks_pipeline.my_project_pipeline,\n"+
		"	on bundle.tf.json line 39, in resource.databricks_pipeline.my_project_pipeline:\n"+
		"	39:       }")).Error()

	expected := "EPERM2: permission denied creating or updating my_project_pipeline.\n" +
		"For assistance, users or groups with appropriate permissions may include: data_team@databricks.com.\n" +
		"They can redeploy the project to apply the latest set of permissions.\n" +
		"Please refer to https://docs.databricks.com/dev-tools/bundles/permissions.html for more on managing permissions."
	require.ErrorContains(t, err, expected)
}
