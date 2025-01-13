package metadata

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/libs/diag"
)

type compute struct{}

func Compute() bundle.Mutator {
	return &compute{}
}

func (m *compute) Name() string {
	return "metadata.Compute"
}

func (m *compute) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	b.Metadata = metadata.Metadata{
		Version: metadata.Version,
		Config:  metadata.Config{},
	}

	// Set Git details in metadata
	b.Metadata.Config.Bundle.Git = config.Git{
		Branch:         b.Config.Bundle.Git.Branch,
		OriginURL:      b.Config.Bundle.Git.OriginURL,
		Commit:         b.Config.Bundle.Git.Commit,
		BundleRootPath: b.Config.Bundle.Git.BundleRootPath,
	}

	// Set job config paths in metadata
	jobsMetadata := make(map[string]*metadata.Job)
	for name, job := range b.Config.Resources.Jobs {
		// Compute config file path the job is defined in, relative to the bundle
		// root
		l := b.Config.GetLocation("resources.jobs." + name)
		relativePath, err := filepath.Rel(b.BundleRootPath, l.File)
		if err != nil {
			return diag.Errorf("failed to compute relative path for job %s: %v", name, err)
		}
		// Metadata for the job
		jobsMetadata[name] = &metadata.Job{
			ID:           job.ID,
			RelativePath: filepath.ToSlash(relativePath),
		}
	}
	b.Metadata.Config.Resources.Jobs = jobsMetadata

	// Set file upload destination of the bundle in metadata
	b.Metadata.Config.Workspace.FilePath = b.Config.Workspace.FilePath
	// In source-linked deployment files are not copied and resources use source files, therefore we use sync path as file path in metadata
	if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		b.Metadata.Config.Workspace.FilePath = b.SyncRootPath
	}
	return nil
}
