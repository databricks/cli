package metadata

import (
	"context"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type annotateJobs struct{}

func AnnotateJobs() bundle.Mutator {
	return &annotateJobs{}
}

func (m *annotateJobs) Name() string {
	return "metadata.AnnotateJobs"
}

func (m *annotateJobs) Apply(_ context.Context, b *bundle.Bundle) error {
	for _, job := range b.Config.Resources.Jobs {
		job.JobSettings.Deployment = &jobs.JobDeployment{
			Kind:             jobs.JobDeploymentKindBundle,
			MetadataFilePath: path.Join(b.Config.Workspace.StatePath, MetadataFileName),
		}
	}

	return nil
}
