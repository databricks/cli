package resourcemutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/require"
)

func TestValidateProductionPipelines(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Pipelines: map[string]*resources.Pipeline{
					"pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
							Development: true,
						},
					},
				},
			},
		},
	}

	diags := validateProductionMode(b, false)

	require.EqualError(t, diags.Error(), "target with 'mode: production' cannot include a pipeline with 'development: true'")
}

func TestProcessTargetModeProduction(t *testing.T) {
	b := mockBundle(config.Production)

	diags := validateProductionMode(b, false)
	require.ErrorContains(t, diags.Error(), "A common practice is to use a username or principal name in this path, i.e. use\n\n  root_path: /Workspace/Users/lennart@company.com/.bundle/${bundle.name}/${bundle.target}")

	b.Config.Workspace.StatePath = "/Shared/.bundle/x/y/state"
	b.Config.Workspace.ArtifactPath = "/Shared/.bundle/x/y/artifacts"
	b.Config.Workspace.FilePath = "/Shared/.bundle/x/y/files"
	b.Config.Workspace.ResourcePath = "/Shared/.bundle/x/y/resources"

	diags = validateProductionMode(b, false)
	require.ErrorContains(t, diags.Error(), "A common practice is to use a username or principal name in this path, i.e. use\n\n  root_path: /Workspace/Users/lennart@company.com/.bundle/${bundle.name}/${bundle.target}")

	jobPermissions := []resources.JobPermission{
		{
			Level:    "CAN_MANAGE",
			UserName: "user@company.com",
		},
	}
	pipelinePermissions := []resources.PipelinePermission{
		{
			Level:    "CAN_MANAGE",
			UserName: "user@company.com",
		},
	}
	experimentPermissions := []resources.MlflowExperimentPermission{
		{
			Level:    "CAN_MANAGE",
			UserName: "user@company.com",
		},
	}
	modelPermissions := []resources.MlflowModelPermission{
		{
			Level:    "CAN_MANAGE",
			UserName: "user@company.com",
		},
	}
	endpointPermissions := []resources.ModelServingEndpointPermission{
		{
			Level:    "CAN_MANAGE",
			UserName: "user@company.com",
		},
	}
	clusterPermissions := []resources.ClusterPermission{
		{
			Level:    "CAN_MANAGE",
			UserName: "user@company.com",
		},
	}
	b.Config.Resources.Jobs["job1"].Permissions = jobPermissions
	b.Config.Resources.Jobs["job1"].RunAs = &jobs.JobRunAs{UserName: "user@company.com"}
	b.Config.Resources.Jobs["job2"].RunAs = &jobs.JobRunAs{UserName: "user@company.com"}
	b.Config.Resources.Jobs["job3"].RunAs = &jobs.JobRunAs{UserName: "user@company.com"}
	b.Config.Resources.Jobs["job4"].RunAs = &jobs.JobRunAs{UserName: "user@company.com"}
	b.Config.Resources.Pipelines["pipeline1"].Permissions = pipelinePermissions
	b.Config.Resources.Experiments["experiment1"].Permissions = experimentPermissions
	b.Config.Resources.Experiments["experiment2"].Permissions = experimentPermissions
	b.Config.Resources.Models["model1"].Permissions = modelPermissions
	b.Config.Resources.ModelServingEndpoints["servingendpoint1"].Permissions = endpointPermissions
	b.Config.Resources.Clusters["cluster1"].Permissions = clusterPermissions

	diags = validateProductionMode(b, false)
	require.NoError(t, diags.Error())

	assert.Equal(t, "job1", b.Config.Resources.Jobs["job1"].Name)
	assert.Equal(t, "pipeline1", b.Config.Resources.Pipelines["pipeline1"].Name)
	assert.False(t, b.Config.Resources.Pipelines["pipeline1"].CreatePipeline.Development)
	assert.Equal(t, "servingendpoint1", b.Config.Resources.ModelServingEndpoints["servingendpoint1"].Name)
	assert.Equal(t, "registeredmodel1", b.Config.Resources.RegisteredModels["registeredmodel1"].Name)
	assert.Equal(t, "qualityMonitor1", b.Config.Resources.QualityMonitors["qualityMonitor1"].TableName)
	assert.Equal(t, "schema1", b.Config.Resources.Schemas["schema1"].Name)
	assert.Equal(t, "volume1", b.Config.Resources.Volumes["volume1"].Name)
	assert.Equal(t, "cluster1", b.Config.Resources.Clusters["cluster1"].ClusterName)
}

func TestProcessTargetModeProductionOkForPrincipal(t *testing.T) {
	b := mockBundle(config.Production)

	// Our target has all kinds of problems when not using service principals ...
	diags := validateProductionMode(b, false)
	require.Error(t, diags.Error())

	// ... but we're much less strict when a principal is used
	diags = validateProductionMode(b, true)
	require.NoError(t, diags.Error())
}

func TestProcessTargetModeProductionOkWithRootPath(t *testing.T) {
	b := mockBundle(config.Production)

	// Our target has all kinds of problems when not using service principals ...
	diags := validateProductionMode(b, false)
	require.Error(t, diags.Error())

	// ... but we're okay if we specify a root path
	b.Target = &config.Target{
		Workspace: &config.Workspace{
			RootPath: "some-root-path",
		},
	}
	diags = validateProductionMode(b, false)
	require.NoError(t, diags.Error())
}

func TestTriggerPauseStatusWhenUnpaused(t *testing.T) {
	b := mockBundle(config.Development)
	b.Config.Presets.TriggerPauseStatus = config.Unpaused

	diags := bundle.ApplySeq(context.Background(), b, ValidateTargetMode())
	require.ErrorContains(t, diags.Error(), "target with 'mode: development' cannot set trigger pause status to UNPAUSED by default")
}

func TestValidateDevelopmentMode(t *testing.T) {
	// Test with a valid development mode bundle
	b := mockBundle(config.Development)
	diags := validateDevelopmentMode(b)
	require.NoError(t, diags.Error())

	// Test with /Volumes path
	b = mockBundle(config.Development)
	b.Config.Workspace.ArtifactPath = "/Volumes/catalog/schema/lennart/libs"
	diags = validateDevelopmentMode(b)
	require.NoError(t, diags.Error())
	b.Config.Workspace.ArtifactPath = "/Volumes/catalog/schema/libs"
	diags = validateDevelopmentMode(b)
	require.ErrorContains(t, diags.Error(), "artifact_path should contain the current username or ${workspace.current_user.short_name} to ensure uniqueness when using 'mode: development'")

	// Test with a bundle that has a non-user path
	b = mockBundle(config.Development)
	b.Config.Workspace.RootPath = "/Shared/.bundle/x/y/state"
	diags = validateDevelopmentMode(b)
	require.ErrorContains(t, diags.Error(), "root_path must start with '~/' or contain the current username to ensure uniqueness when using 'mode: development'")

	// Test with a bundle that has an unpaused trigger pause status
	b = mockBundle(config.Development)
	b.Config.Presets.TriggerPauseStatus = config.Unpaused
	diags = validateDevelopmentMode(b)
	require.ErrorContains(t, diags.Error(), "UNPAUSED")

	// Test with a bundle that has a prefix not containing the username or short name
	b = mockBundle(config.Development)
	b.Config.Presets.NamePrefix = "[prod]"
	diags = validateDevelopmentMode(b)
	require.Len(t, diags, 1)
	assert.Equal(t, diag.Error, diags[0].Severity)
	assert.Contains(t, diags[0].Summary, "")

	// Test with a bundle that has valid user paths
	b = mockBundle(config.Development)
	b.Config.Workspace.RootPath = "/Users/lennart@company.com/.bundle/x/y/state"
	b.Config.Workspace.StatePath = "/Users/lennart@company.com/.bundle/x/y/state"
	b.Config.Workspace.FilePath = "/Users/lennart@company.com/.bundle/x/y/files"
	b.Config.Workspace.ArtifactPath = "/Users/lennart@company.com/.bundle/x/y/artifacts"
	b.Config.Workspace.ResourcePath = "/Users/lennart@company.com/.bundle/x/y/resources"
	diags = validateDevelopmentMode(b)
	require.NoError(t, diags.Error())
}
