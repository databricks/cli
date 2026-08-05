package metadata

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/pipelines"
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
							Deployment: &jobs.JobDeployment{Kind: jobs.JobDeploymentKindBundle},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
							Deployment: &pipelines.PipelineDeployment{Kind: pipelines.DeploymentKindBundle},
						},
					},
				},
			},
		},
	}

	diags := bundle.ApplySeq(t.Context(), b, AnnotateDeploymentVersion("dep-123", 7))
	require.NoError(t, diags.Error())

	job := b.Config.Resources.Jobs["my-job"].Deployment
	assert.Equal(t, "dep-123", job.DeploymentId)
	assert.Equal(t, "7", job.VersionId)
	// The kind set by AnnotateJobs is preserved.
	assert.Equal(t, jobs.JobDeploymentKindBundle, job.Kind)

	pipeline := b.Config.Resources.Pipelines["my-pipeline"].Deployment
	assert.Equal(t, "dep-123", pipeline.DeploymentId)
	assert.Equal(t, "7", pipeline.VersionId)
	assert.Equal(t, pipelines.DeploymentKindBundle, pipeline.Kind)
}
