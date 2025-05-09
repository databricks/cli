package resourcemutator

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle/permissions"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
)

var jobAlice = resources.JobPermission{
	Level:    "CAN_MANAGE",
	UserName: "alice@databricks.com",
}

var jobBob = resources.JobPermission{
	Level:    "CAN_VIEW",
	UserName: "bob@databricks.com",
}

var jobRobot = resources.JobPermission{
	Level:                "CAN_MANAGE_RUN",
	ServicePrincipalName: "i-Robot",
}

var pipelineAlice = resources.PipelinePermission{
	Level:    permissions.CAN_MANAGE,
	UserName: "alice@databricks.com",
}

var pipelineBob = resources.PipelinePermission{
	Level:    permissions.CAN_VIEW,
	UserName: "bob@databricks.com",
}

var pipelineRobot = resources.PipelinePermission{
	Level:                permissions.CAN_RUN,
	ServicePrincipalName: "i-Robot",
}

var experimentAlice = resources.MlflowExperimentPermission{
	Level:    permissions.CAN_MANAGE,
	UserName: "alice@databricks.com",
}

var experimentBob = resources.MlflowExperimentPermission{
	Level:    permissions.CAN_VIEW,
	UserName: "bob@databricks.com",
}

var experimentRobot = resources.MlflowExperimentPermission{
	Level:                permissions.CAN_RUN,
	ServicePrincipalName: "i-Robot",
}

var modelAlice = resources.MlflowModelPermission{
	Level:    permissions.CAN_MANAGE,
	UserName: "alice@databricks.com",
}

var modelBob = resources.MlflowModelPermission{
	Level:    permissions.CAN_VIEW,
	UserName: "bob@databricks.com",
}

var modelRobot = resources.MlflowModelPermission{
	Level:                permissions.CAN_RUN,
	ServicePrincipalName: "i-Robot",
}

var endpointAlice = resources.ModelServingEndpointPermission{
	Level:    permissions.CAN_MANAGE,
	UserName: "alice@databricks.com",
}

var endpointBob = resources.ModelServingEndpointPermission{
	Level:    permissions.CAN_VIEW,
	UserName: "bob@databricks.com",
}

var endpointRobot = resources.ModelServingEndpointPermission{
	Level:                permissions.CAN_RUN,
	ServicePrincipalName: "i-Robot",
}

func testFixture(userName string) *bundle.Bundle {
	jobPermissions := []resources.JobPermission{
		jobAlice,
		jobBob,
		jobRobot,
	}

	pipelinePermissions := []resources.PipelinePermission{
		pipelineAlice,
		pipelineBob,
		pipelineRobot,
	}

	experimentPermissions := []resources.MlflowExperimentPermission{
		experimentAlice,
		experimentBob,
		experimentRobot,
	}

	modelPermissions := []resources.MlflowModelPermission{
		modelAlice,
		modelBob,
		modelRobot,
	}

	endpointPermissions := []resources.ModelServingEndpointPermission{
		endpointAlice,
		endpointBob,
		endpointRobot,
	}

	return &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: userName,
					},
				},
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"job1": {
						JobSettings: jobs.JobSettings{
							Name: "job1",
						},
						Permissions: jobPermissions,
					},
					"job2": {
						JobSettings: jobs.JobSettings{
							Name: "job2",
						},
						Permissions: jobPermissions,
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {
						Permissions: pipelinePermissions,
					},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment1": {
						Permissions: experimentPermissions,
					},
				},
				Models: map[string]*resources.MlflowModel{
					"model1": {
						Permissions: modelPermissions,
					},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint1": {
						Permissions: endpointPermissions,
					},
				},
				RegisteredModels: map[string]*resources.RegisteredModel{
					"registered_model1": {
						Grants: []resources.Grant{
							{
								Principal: "abc",
							},
						},
					},
				},
			},
		},
	}
}

func TestFilterCurrentUser(t *testing.T) {
	b := testFixture("alice@databricks.com")

	diags := bundle.Apply(context.Background(), b, FilterCurrentUser())
	assert.NoError(t, diags.Error())

	// Assert current user is filtered out.
	assert.Len(t, b.Config.Resources.Jobs["job1"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.Jobs["job1"].Permissions, jobRobot)
	assert.Contains(t, b.Config.Resources.Jobs["job1"].Permissions, jobBob)

	assert.Len(t, b.Config.Resources.Jobs["job2"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.Jobs["job2"].Permissions, jobRobot)
	assert.Contains(t, b.Config.Resources.Jobs["job2"].Permissions, jobBob)

	assert.Len(t, b.Config.Resources.Pipelines["pipeline1"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.Pipelines["pipeline1"].Permissions, pipelineRobot)
	assert.Contains(t, b.Config.Resources.Pipelines["pipeline1"].Permissions, pipelineBob)

	assert.Len(t, b.Config.Resources.Experiments["experiment1"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Permissions, experimentRobot)
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Permissions, experimentBob)

	assert.Len(t, b.Config.Resources.Models["model1"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.Models["model1"].Permissions, modelRobot)
	assert.Contains(t, b.Config.Resources.Models["model1"].Permissions, modelBob)

	assert.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions, endpointRobot)
	assert.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions, endpointBob)

	// Assert there's no change to the grant.
	assert.Len(t, b.Config.Resources.RegisteredModels["registered_model1"].Grants, 1)
}

func TestFilterCurrentServicePrincipal(t *testing.T) {
	b := testFixture("i-Robot")

	diags := bundle.Apply(context.Background(), b, FilterCurrentUser())
	assert.NoError(t, diags.Error())

	// Assert current user is filtered out.
	assert.Len(t, b.Config.Resources.Jobs["job1"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.Jobs["job1"].Permissions, jobAlice)
	assert.Contains(t, b.Config.Resources.Jobs["job1"].Permissions, jobBob)

	assert.Len(t, b.Config.Resources.Jobs["job2"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.Jobs["job2"].Permissions, jobAlice)
	assert.Contains(t, b.Config.Resources.Jobs["job2"].Permissions, jobBob)

	assert.Len(t, b.Config.Resources.Pipelines["pipeline1"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.Pipelines["pipeline1"].Permissions, pipelineAlice)
	assert.Contains(t, b.Config.Resources.Pipelines["pipeline1"].Permissions, pipelineBob)

	assert.Len(t, b.Config.Resources.Experiments["experiment1"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Permissions, experimentAlice)
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Permissions, experimentBob)

	assert.Len(t, b.Config.Resources.Models["model1"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.Models["model1"].Permissions, modelAlice)
	assert.Contains(t, b.Config.Resources.Models["model1"].Permissions, modelBob)

	assert.Len(t, b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions, 2)
	assert.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions, endpointAlice)
	assert.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions, endpointBob)

	// Assert there's no change to the grant.
	assert.Len(t, b.Config.Resources.RegisteredModels["registered_model1"].Grants, 1)
}

func TestFilterCurrentUserDoesNotErrorWhenNoResources(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				CurrentUser: &config.User{
					User: &iam.User{
						UserName: "abc",
					},
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, FilterCurrentUser())
	assert.NoError(t, diags.Error())
}
