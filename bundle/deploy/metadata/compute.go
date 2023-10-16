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
		Version: deploy.MetadataVersion,
		Config:  config.Root{},
	}

	// Set git details in metadata
	b.Metadata.Config.Bundle.Git = b.Config.Bundle.Git

	// Set job config paths in metadata
	jobsMetadata := make(map[string]*resources.Job)
	for name, job := range b.Config.Resources.Jobs {
		// Compute config file path the job is defined in, relative to the bundle
		// root
		relativePath, err := filepath.Rel(b.Config.Path, job.ConfigFilePath)
		if err != nil {
			return fmt.Errorf("failed to compute relative path for job %s: %w", name, err)
		}
		relativePath = filepath.ToSlash(relativePath)

		// Metadata for the job
		jobsMetadata[name] = &resources.Job{
			ID: job.ID,
			Paths: paths.Paths{
				RelativePath: path.Clean(relativePath),
			},
		}
	}
	b.Metadata.Config.Resources.Jobs = jobsMetadata

	// Set file upload destination of the bundle in metadata
	b.Metadata.Config.Workspace.FilesPath = b.Config.Workspace.FilesPath
	return nil
}
