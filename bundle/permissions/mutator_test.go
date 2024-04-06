package permissions

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/require"
)

func TestApplyBundlePermissions(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.com",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "TestUser"},
				{Level: CAN_VIEW, GroupName: "TestGroup"},
				{Level: CAN_RUN, ServicePrincipalName: "TestServicePrincipal"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {},
					"job_2": {},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline_1": {},
					"pipeline_2": {},
				},
				Models: map[string]*resources.MlflowModel{
					"model_1": {},
					"model_2": {},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment_1": {},
					"experiment_2": {},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint_1": {},
					"endpoint_2": {},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	require.Len(t, b.Config.Resources.Jobs["job_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Jobs["job_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.Permission{Level: "CAN_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.Permission{Level: "CAN_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Models["model_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Models["model_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Experiments["experiment_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Experiments["experiment_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.Permission{Level: "CAN_QUERY", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.Permission{Level: "CAN_QUERY", ServicePrincipalName: "TestServicePrincipal"})
}

func TestWarningOnOverlapPermission(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.com",
			},
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "TestUser"},
				{Level: CAN_VIEW, GroupName: "TestGroup"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {
						Permissions: []resources.Permission{
							{Level: CAN_VIEW, UserName: "TestUser"},
						},
					},
					"job_2": {
						Permissions: []resources.Permission{
							{Level: CAN_VIEW, UserName: "TestUser2"},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_VIEW", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_VIEW", UserName: "TestUser2"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})

}
