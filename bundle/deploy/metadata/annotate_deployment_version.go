package metadata

import (
	"context"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type annotateDeploymentVersion struct {
	deploymentID string
	version      int64
}

// AnnotateDeploymentVersion stamps the DMS deployment and version onto every job
// and pipeline, so a resource in the workspace points back at the deployment that
// produced it (which is how lineage resolves a job to its bundle).
//
// AnnotateJobs/AnnotatePipelines set the rest of the deployment metadata during
// initialize, but the version - and, on a first deploy, the deployment ID - only
// exist once CreateVersion has run, so these two fields are stamped separately
// from the deploy phase.
func AnnotateDeploymentVersion(deploymentID string, version int64) bundle.Mutator {
	return &annotateDeploymentVersion{deploymentID: deploymentID, version: version}
}

func (m *annotateDeploymentVersion) Name() string {
	return "metadata.AnnotateDeploymentVersion"
}

func (m *annotateDeploymentVersion) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	versionID := strconv.FormatInt(m.version, 10)

	for _, job := range b.Config.Resources.Jobs {
		// Deployment is set by AnnotateJobs, which runs during initialize.
		job.Deployment.DeploymentId = m.deploymentID
		job.Deployment.VersionId = versionID
	}

	for _, pipeline := range b.Config.Resources.Pipelines {
		pipeline.Deployment.DeploymentId = m.deploymentID
		pipeline.Deployment.VersionId = versionID
	}

	return nil
}
