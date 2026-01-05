package metadata

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

type compute struct{}

func Compute() bundle.Mutator {
	return &compute{}
}

func (m *compute) Name() string {
	return "metadata.Compute"
}

func (m *compute) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
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

	// Set bundle name, target, and mode
	b.Metadata.Config.Bundle.Name = b.Config.Bundle.Name
	b.Metadata.Config.Bundle.Target = b.Config.Bundle.Target
	b.Metadata.Config.Bundle.Mode = string(b.Config.Bundle.Mode)

	// Set job config paths in metadata
	jobsMetadata := make(map[string]*metadata.Resource)
	for name, job := range b.Config.Resources.Jobs {
		// Compute config file path the job is defined in, relative to the bundle
		// root
		l := b.Config.GetLocation("resources.jobs." + name)
		if l.File == "" {
			// b.Config.Resources.Jobs may include a job that only exists in state but not in config
			continue
		}

		relativePath, err := filepath.Rel(b.BundleRootPath, l.File)
		if err != nil {
			log.Warnf(ctx, "failed to compute relative path for job %q: %s", name, err)
			relativePath = ""
		}

		// Metadata for the job
		jobsMetadata[name] = &metadata.Resource{
			ID:           job.ID,
			RelativePath: filepath.ToSlash(relativePath),
		}
	}
	b.Metadata.Config.Resources.Jobs = jobsMetadata

	// Set pipeline config paths in metadata
	pipelinesMetadata := make(map[string]*metadata.Resource)
	for name, pipeline := range b.Config.Resources.Pipelines {
		// Compute config file path the pipeline is defined in, relative to the bundle
		// root
		l := b.Config.GetLocation("resources.pipelines." + name)
		relativePath, err := filepath.Rel(b.BundleRootPath, l.File)
		if err != nil {
			return diag.Errorf("failed to compute relative path for pipeline %s: %v", name, err)
		}
		// Metadata for the pipeline
		pipelinesMetadata[name] = &metadata.Resource{
			ID:           pipeline.ID,
			RelativePath: filepath.ToSlash(relativePath),
		}
	}
	b.Metadata.Config.Resources.Pipelines = pipelinesMetadata

	// Set file upload destination of the bundle in metadata
	b.Metadata.Config.Workspace.FilePath = b.Config.Workspace.FilePath
	// In source-linked deployment files are not copied and resources use source files, therefore we use sync path as file path in metadata
	if config.IsExplicitlyEnabled(b.Config.Presets.SourceLinkedDeployment) {
		b.Metadata.Config.Workspace.FilePath = b.SyncRootPath
		b.Metadata.Config.Presets.SourceLinkedDeployment = true
	}

	// Set the git folder path for deployments from the workspace
	if b.WorktreeRoot != nil && strings.HasPrefix(b.WorktreeRoot.Native(), "/Workspace/") {
		b.Metadata.Extra.GitFolderPath = b.WorktreeRoot.Native()
	}

	return nil
}
