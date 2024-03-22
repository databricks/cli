package permissions

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/iam"
	"github.com/stretchr/testify/assert"
)

var alice = resources.Permission{
	Level:    CAN_MANAGE,
	UserName: "alice@databricks.com",
}

var bob = resources.Permission{
	Level:    CAN_VIEW,
	UserName: "bob@databricks.com",
}

var robot = resources.Permission{
	Level:                CAN_RUN,
	ServicePrincipalName: "i-Robot",
}

func testFixture(userName string) *bundle.Bundle {
	p := []resources.Permission{
		alice,
		bob,
		robot,
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
						Permissions: p,
					},
					"job2": {
						Permissions: p,
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"pipeline1": {
						Permissions: p,
					},
				},
				Experiments: map[string]*resources.MlflowExperiment{
					"experiment1": {
						Permissions: p,
					},
				},
				Models: map[string]*resources.MlflowModel{
					"model1": {
						Permissions: p,
					},
				},
				ModelServingEndpoints: map[string]*resources.ModelServingEndpoint{
					"endpoint1": {
						Permissions: p,
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
	assert.Empty(t, diags)

	// Assert current user is filtered out.
	assert.Equal(t, 2, len(b.Config.Resources.Jobs["job1"].Permissions))
	assert.Contains(t, b.Config.Resources.Jobs["job1"].Permissions, robot)
	assert.Contains(t, b.Config.Resources.Jobs["job1"].Permissions, bob)

	assert.Equal(t, 2, len(b.Config.Resources.Jobs["job2"].Permissions))
	assert.Contains(t, b.Config.Resources.Jobs["job2"].Permissions, robot)
	assert.Contains(t, b.Config.Resources.Jobs["job2"].Permissions, bob)

	assert.Equal(t, 2, len(b.Config.Resources.Pipelines["pipeline1"].Permissions))
	assert.Contains(t, b.Config.Resources.Pipelines["pipeline1"].Permissions, robot)
	assert.Contains(t, b.Config.Resources.Pipelines["pipeline1"].Permissions, bob)

	assert.Equal(t, 2, len(b.Config.Resources.Experiments["experiment1"].Permissions))
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Permissions, robot)
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Permissions, bob)

	assert.Equal(t, 2, len(b.Config.Resources.Models["model1"].Permissions))
	assert.Contains(t, b.Config.Resources.Models["model1"].Permissions, robot)
	assert.Contains(t, b.Config.Resources.Models["model1"].Permissions, bob)

	assert.Equal(t, 2, len(b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions))
	assert.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions, robot)
	assert.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions, bob)

	// Assert there's no change to the grant.
	assert.Equal(t, 1, len(b.Config.Resources.RegisteredModels["registered_model1"].Grants))
}

func TestFilterCurrentServicePrincipal(t *testing.T) {
	b := testFixture("i-Robot")

	diags := bundle.Apply(context.Background(), b, FilterCurrentUser())
	assert.Empty(t, diags)

	// Assert current user is filtered out.
	assert.Equal(t, 2, len(b.Config.Resources.Jobs["job1"].Permissions))
	assert.Contains(t, b.Config.Resources.Jobs["job1"].Permissions, alice)
	assert.Contains(t, b.Config.Resources.Jobs["job1"].Permissions, bob)

	assert.Equal(t, 2, len(b.Config.Resources.Jobs["job2"].Permissions))
	assert.Contains(t, b.Config.Resources.Jobs["job2"].Permissions, alice)
	assert.Contains(t, b.Config.Resources.Jobs["job2"].Permissions, bob)

	assert.Equal(t, 2, len(b.Config.Resources.Pipelines["pipeline1"].Permissions))
	assert.Contains(t, b.Config.Resources.Pipelines["pipeline1"].Permissions, alice)
	assert.Contains(t, b.Config.Resources.Pipelines["pipeline1"].Permissions, bob)

	assert.Equal(t, 2, len(b.Config.Resources.Experiments["experiment1"].Permissions))
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Permissions, alice)
	assert.Contains(t, b.Config.Resources.Experiments["experiment1"].Permissions, bob)

	assert.Equal(t, 2, len(b.Config.Resources.Models["model1"].Permissions))
	assert.Contains(t, b.Config.Resources.Models["model1"].Permissions, alice)
	assert.Contains(t, b.Config.Resources.Models["model1"].Permissions, bob)

	assert.Equal(t, 2, len(b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions))
	assert.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions, alice)
	assert.Contains(t, b.Config.Resources.ModelServingEndpoints["endpoint1"].Permissions, bob)

	// Assert there's no change to the grant.
	assert.Equal(t, 1, len(b.Config.Resources.RegisteredModels["registered_model1"].Grants))
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
	assert.Empty(t, diags)

}
