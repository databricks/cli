package metadata

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotateResourcesAppliesRegisteredAnnotations(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Workspace: config.Workspace{
				StatePath: "/a/b/c",
			},
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job": {},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline": {},
				},
			},
		},
	}

	ctx := dbr.MockRuntime(t.Context(), dbr.Environment{IsDbr: false, Version: ""})
	diags := bundle.Apply(ctx, b, AnnotateResources())
	require.NoError(t, diags.Error())

	assert.Equal(t, jobs.JobDeploymentKindBundle, b.Config.Resources.Jobs["my-job"].Deployment.Kind)
	assert.Equal(t, pipelines.DeploymentKindBundle, b.Config.Resources.Pipelines["my-pipeline"].Deployment.Kind)
}
