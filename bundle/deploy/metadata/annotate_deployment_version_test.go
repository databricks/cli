package metadata

import (
	"testing"

	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"github.com/databricks/cli/libs/sdktypes/pipelines/v2"
	"github.com/databricks/cli/libs/utils"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnnotateDeploymentVersion(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job": {
						JobSettings: jobs.JobSettings{
							Deployment: &jobs.JobDeployment{Kind: jobs.JobDeployment_DeploymentKind_Bundle},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline": {
						CreatePipelineRequest: pipelines.CreatePipelineRequest{
							Deployment: &pipelines.PipelineDeployment{Kind: pipelines.DeploymentKind_Bundle},
						},
					},
				},
			},
		},
	}

	diags := bundle.ApplySeq(t.Context(), b, AnnotateDeployment("dep-123"), AnnotateDeploymentVersion(7))
	require.NoError(t, diags.Error())

	job := b.Config.Resources.Jobs["my-job"].Deployment
	assert.Equal(t, "dep-123", utils.Deref(job.DeploymentId))
	assert.Equal(t, "7", utils.Deref(job.VersionId))
	// The kind set by AnnotateJobs is preserved.
	assert.Equal(t, jobs.JobDeployment_DeploymentKind_Bundle, job.Kind)

	pipeline := b.Config.Resources.Pipelines["my-pipeline"].Deployment
	assert.Equal(t, "dep-123", utils.Deref(pipeline.DeploymentId))
	assert.Equal(t, "7", utils.Deref(pipeline.VersionId))
	assert.Equal(t, pipelines.DeploymentKind_Bundle, pipeline.Kind)
}
