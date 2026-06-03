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

func bundleWithDeploymentBlocks() *bundle.Bundle {
	return &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{
					"my-job": {
						JobSettings: jobs.JobSettings{
							Name:       "My Job",
							Deployment: &jobs.JobDeployment{Kind: jobs.JobDeploymentKindBundle, MetadataFilePath: "/a/b/c/metadata.json"},
						},
					},
				},
				Pipelines: map[string]*resources.Pipeline{
					"my-pipeline": {
						CreatePipeline: pipelines.CreatePipeline{
							Name:       "My Pipeline",
							Deployment: &pipelines.PipelineDeployment{Kind: pipelines.DeploymentKindBundle, MetadataFilePath: "/a/b/c/metadata.json"},
						},
					},
				},
			},
		},
	}
}

func TestAnnotateDeploymentVersionStampsIDs(t *testing.T) {
	b := bundleWithDeploymentBlocks()
	b.DeploymentID = "dep-123"
	b.DeploymentVersionID = "7"

	diags := AnnotateDeploymentVersion().Apply(t.Context(), b)
	require.NoError(t, diags.Error())

	job := b.Config.Resources.Jobs["my-job"].Deployment
	assert.Equal(t, "dep-123", job.DeploymentId)
	assert.Equal(t, "7", job.VersionId)
	// Existing fields are preserved.
	assert.Equal(t, jobs.JobDeploymentKindBundle, job.Kind)
	assert.Equal(t, "/a/b/c/metadata.json", job.MetadataFilePath)

	pipeline := b.Config.Resources.Pipelines["my-pipeline"].Deployment
	assert.Equal(t, "dep-123", pipeline.DeploymentId)
	assert.Equal(t, "7", pipeline.VersionId)
	assert.Equal(t, pipelines.DeploymentKindBundle, pipeline.Kind)
	assert.Equal(t, "/a/b/c/metadata.json", pipeline.MetadataFilePath)
}

func TestAnnotateDeploymentVersionNoopWithoutDMS(t *testing.T) {
	b := bundleWithDeploymentBlocks()
	// DeploymentID is empty: the deployment metadata service is not in use.

	diags := AnnotateDeploymentVersion().Apply(t.Context(), b)
	require.NoError(t, diags.Error())

	assert.Empty(t, b.Config.Resources.Jobs["my-job"].Deployment.DeploymentId)
	assert.Empty(t, b.Config.Resources.Jobs["my-job"].Deployment.VersionId)
	assert.Empty(t, b.Config.Resources.Pipelines["my-pipeline"].Deployment.DeploymentId)
	assert.Empty(t, b.Config.Resources.Pipelines["my-pipeline"].Deployment.VersionId)
}
