package metadata

import (
	"context"
	"strconv"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type annotateDeployment struct {
	deploymentID string
}

// AnnotateDeployment stamps the DMS deployment onto every job and pipeline, so a workspace
// resource points back at the deployment that produced it - how lineage resolves a job to its
// bundle. It runs before the plan, or the stamp would show as drift against local config.
func AnnotateDeployment(deploymentID string) bundle.Mutator {
	return &annotateDeployment{deploymentID: deploymentID}
}

func (m *annotateDeployment) Name() string {
	return "metadata.AnnotateDeployment"
}

func (m *annotateDeployment) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	for _, job := range b.Config.Resources.Jobs {
		// Deployment is set by AnnotateJobs, which runs during initialize.
		job.Deployment.DeploymentId = m.deploymentID
	}

	for _, pipeline := range b.Config.Resources.Pipelines {
		pipeline.Deployment.DeploymentId = m.deploymentID
	}

	return nil
}

type annotateDeploymentVersion struct {
	version int64
}

// AnnotateDeploymentVersion stamps the DMS version onto every job and pipeline.
// Separate from AnnotateDeployment because version only exists after CreateVersion runs.
func AnnotateDeploymentVersion(version int64) bundle.Mutator {
	return &annotateDeploymentVersion{version: version}
}

func (m *annotateDeploymentVersion) Name() string {
	return "metadata.AnnotateDeploymentVersion"
}

func (m *annotateDeploymentVersion) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	versionID := strconv.FormatInt(m.version, 10)

	for _, job := range b.Config.Resources.Jobs {
		job.Deployment.VersionId = versionID
	}

	for _, pipeline := range b.Config.Resources.Pipelines {
		pipeline.Deployment.VersionId = versionID
	}

	return nil
}
