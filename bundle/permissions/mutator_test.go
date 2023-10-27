package permissions

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/ml"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/databricks/databricks-sdk-go/service/serving"
	"github.com/stretchr/testify/require"
)

func TestApplyTopLevelPermission(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Permissions: []resources.Permission{
				{Level: CAN_MANAGE, UserName: "TestUser"},
				{Level: CAN_VIEW, GroupName: "TestGroup"},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job_1": {JobSettings: &jobs.JobSettings{}},
					"job_2": {JobSettings: &jobs.JobSettings{}},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline_1": {PipelineSpec: &pipelines.PipelineSpec{}},
					"pipeline_2": {PipelineSpec: &pipelines.PipelineSpec{}},
				},
				Models: map[string]*resources.MlflowModel{
					"model_1": {Model: &ml.Model{}},
					"model_2": {Model: &ml.Model{}},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment_1": {Experiment: &ml.Experiment{}},
					"experiment_2": {Experiment: &ml.Experiment{}},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint_1": {CreateServingEndpoint: &serving.CreateServingEndpoint{}},
					"endpoint_2": {CreateServingEndpoint: &serving.CreateServingEndpoint{}},
				},
			},
		},
	}

	err := bundle.Apply(context.Background(), b, ApplyTopLevelPermissions())
	require.NoError(t, err)

	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})

	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Pipelines["pipeline_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})

	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_1"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Models["model_2"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_1"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Experiments["experiment_2"].Permissions, resources.Permission{Level: "CAN_READ", GroupName: "TestGroup"})

	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
}

func TestWarningOnOverlapPermission(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
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
						JobSettings: &jobs.JobSettings{},
					},
					"job_2": {
						Permissions: []resources.Permission{
							{Level: CAN_VIEW, UserName: "TestUser2"},
						},
						JobSettings: &jobs.JobSettings{},
					},
				},
			},
		},
	}

	err := bundle.Apply(context.Background(), b, ApplyTopLevelPermissions())
	require.NoError(t, err)

	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_VIEW", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_1"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_VIEW", UserName: "TestUser2"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_MANAGE", UserName: "TestUser"})
	require.Contains(t, b.Config.Resources.Jobs["job_2"].Permissions, resources.Permission{Level: "CAN_VIEW", GroupName: "TestGroup"})

}
