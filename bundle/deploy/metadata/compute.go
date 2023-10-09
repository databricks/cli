package metadata

import (
	"context"
	"fmt"
	"path"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/paths"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/deploy"
)

type compute struct{}

func Compute() bundle.Mutator {
	return &compute{}
}

func (m *compute) Name() string {
	return "metadata.Compute"
}

func (m *compute) Apply(_ context.Context, b *bundle.Bundle) error {
	b.Metadata = deploy.Metadata{
		Version: deploy.LatestMetadataVersion,
		Config:  config.Root{},
	}

	// Set git details in metadata
	b.Metadata.Config.Bundle.Git = b.Config.Bundle.Git

	// Set Job paths in metadata
	jobsMetadata := make(map[string]*resources.Job)
	for name, job := range b.Config.Resources.Jobs {
		relativePath, err := filepath.Rel(b.Config.Path, job.ConfigFilePath)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for job %s: %w", name, err)
		}
		relativePath = filepath.ToSlash(relativePath)

		jobsMetadata[name] = &resources.Job{
			Paths: paths.Paths{
				ConfigFilePath: path.Clean(relativePath),
			},
		}
	}
	b.Metadata.Config.Resources.Jobs = jobsMetadata

	// Set root path of the bundle in metadata
	b.Metadata.Config.Workspace.RootPath = b.Config.Workspace.RootPath
	return nil
}
