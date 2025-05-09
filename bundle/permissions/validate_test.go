package permissions

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

func TestValidateSharedRootPermissionsForShared(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Workspace/Shared/foo/bar",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, GroupName: "users"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {JobSettings: jobs.JobSettings{Name: "job_1"}},
					"job_2": {JobSettings: jobs.JobSettings{Name: "job_2"}},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	diags := bundle.Apply(context.Background(), b, ValidateSharedRootPermissions())
	require.Empty(t, diags)
}

func TestValidateSharedRootPermissionsForSharedError(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Workspace/Shared/foo/bar",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "foo@bar.com"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {JobSettings: jobs.JobSettings{Name: "job_1"}},
					"job_2": {JobSettings: jobs.JobSettings{Name: "job_2"}},
				},
			},
		},
	}

	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)

	diags := bundle.Apply(context.Background(), b, ValidateSharedRootPermissions())
	require.Len(t, diags, 1)
	require.Equal(t, "the bundle root path /Workspace/Shared/foo/bar is writable by all workspace users", diags[0].Summary)
	require.Equal(t, diag.Warning, diags[0].Severity)
}
