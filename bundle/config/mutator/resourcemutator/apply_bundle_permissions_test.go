package resourcemutator

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/databricks/cli/bundle/permissions"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyBundlePermissions(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.com",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "TestUser"},
				{Level: permissions.CAN_VIEW, GroupName: "TestGroup"},
				{Level: permissions.CAN_RUN, ServicePrincipalName: "TestServicePrincipal"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {
						JobSettings: jobs.JobSettings{
							Name: "job_1",
						},
					},
					"job_2": {
						JobSettings: jobs.JobSettings{
							Name: "job_2",
						},
					},
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
				Dashboards: map[string]*resources.Dashboard{
					"dashboard_1": {},
					"dashboard_2": {},
				},
				Apps: map[string]*resources.App{
					"app_1": {},
					"app_2": {},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	require.Len(t, b.Config.Resources.Jobs["job_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Jobs["job_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_MANAGE_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.PipelinePermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.PipelinePermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.PipelinePermission{Level: "CAN_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.PipelinePermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.PipelinePermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.PipelinePermission{Level: "CAN_RUN", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Models["model_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.MlflowModelPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.MlflowModelPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Models["model_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.MlflowModelPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.MlflowModelPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Experiments["experiment_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Experiments["experiment_2"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.MlflowExperimentPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, 3)
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_QUERY", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, 3)
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.ModelServingEndpointPermission{Level: "CAN_QUERY", ServicePrincipalName: "TestServicePrincipal"})

	require.Len(t, b.Config.Resources.Dashboards["dashboard_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Dashboards["dashboard_1"].Permissions, resources.DashboardPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Dashboards["dashboard_1"].Permissions, resources.DashboardPermission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Len(t, b.Config.Resources.Apps["app_1"].Permissions, 2)
	require.Contains(t, b.Config.Resources.Apps["app_1"].Permissions, resources.AppPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Apps["app_1"].Permissions, resources.AppPermission{Level: "CAN_USE", GroupName: "TestGroup"})
}

func TestWarningOnOverlapPermission(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				RootPath: "/Users/foo@bar.com",
			},
			Permissions: []resources.Permission{
				{Level: permissions.CAN_MANAGE, UserName: "TestUser"},
				{Level: permissions.CAN_VIEW, GroupName: "TestGroup"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {
						JobSettings: jobs.JobSettings{
							Name: "job_1",
						},
						Permissions: []resources.JobPermission{
							{Level: "CAN_VIEW", UserName: "TestUser"},
						},
					},
					"job_2": {
						JobSettings: jobs.JobSettings{
							Name: "job_2",
						},
						Permissions: []resources.JobPermission{
							{Level: "CAN_VIEW", UserName: "TestUser2"},
						},
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, ApplyBundlePermissions())
	require.NoError(t, diags.Error())

	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_VIEW", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_VIEW", UserName: "TestUser2"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.JobPermission{Level: "CAN_VIEW", GroupName: "TestGroup"})
}

func TestAllResourcesExplicitlyDefinedForPermissionsSupport(t *testing.T) {
	r := config.Resources{}

	for _, resource := range unsupportedResources {
		_, ok := levelsMap[resource]
		assert.False(t, ok, "Resource %s is defined in both levelsMap and unsupportedResources", resource)
	}

	for _, resource := range r.AllResources() {
		_, ok := levelsMap[resource.Description.PluralName]
		if !slices.Contains(unsupportedResources, resource.Description.PluralName) && !ok {
			assert.Fail(t, fmt.Sprintf("Resource %s is not explicitly defined in levelsMap or unsupportedResources", resource.Description.PluralName))
		}
	}
}
