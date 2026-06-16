package metadata

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type annotateDeploymentVersion struct{}

// AnnotateDeploymentVersion records the DMS deployment_id and version_id on the
// deployment block of every job and pipeline. Unlike AnnotateJobs/AnnotatePipelines
// (which run during Initialize), these IDs are only known after the deployment
// lock is acquired, so this mutator runs in the deploy phase. It is a no-op when
// the deployment metadata service is not enabled (DeploymentID is empty).
func AnnotateDeploymentVersion() bundle.Mutator {
	return &annotateDeploymentVersion{}
}

func (m *annotateDeploymentVersion) Name() string {
	return "metadata.AnnotateDeploymentVersion"
}

func (m *annotateDeploymentVersion) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.DeploymentID == "" {
		return nil
	}

	// AnnotateJobs and AnnotatePipelines run during Initialize and always set
	// the Deployment block, so it is non-nil here.
	for _, job := range b.Config.Resources.Jobs {
		job.Deployment.DeploymentId = b.DeploymentID
		job.Deployment.VersionId = b.DeploymentVersionID
	}
	for _, pipeline := range b.Config.Resources.Pipelines {
		pipeline.Deployment.DeploymentId = b.DeploymentID
		pipeline.Deployment.VersionId = b.DeploymentVersionID
	}

	return nil
}
