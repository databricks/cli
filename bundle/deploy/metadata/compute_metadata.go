package metadata

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy"
)

type computeMetadata struct{}

func ComputeMetadata() bundle.Mutator {
	return &computeMetadata{}
}

func (m *computeMetadata) Name() string {
	return "ComputeMetadata"
}

func (m *computeMetadata) Apply(_ context.Context, b *bundle.Bundle) error {
	b.Metadata = deploy.Metadata{
		Version: deploy.LatestMetadataVersion,
		Config:  config.Root{},
	}

	// Set git details in metadata
	b.Metadata.Config.Bundle.Git = b.Config.Bundle.Git

	// Set Job paths in metadata
	jobsMetadata := make(map[string]*resources.Job)
	for name, job := range b.Config.Resources.Jobs {
		jobsMetadata[name] = &resources.Job{
			Paths: paths.Paths{
				ConfigFilePath: job.ConfigFilePath,
			},
		}
	}
	b.Metadata.Config.Resources.Jobs = jobsMetadata
	return nil
}
