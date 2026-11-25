package metadata

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

func init() {
	resourceMetadata.Register("jobs", computeJobs)
}

func computeJobs(ctx context.Context, b *bundle.Bundle, _ resourcestate.ExportedResourcesMap) diag.Diagnostics {
	// Set job config paths in metadata
	jobsMetadata := make(map[string]*metadata.Resource)
	for name, job := range b.Config.Resources.Jobs {
		// Compute config file path the job is defined in, relative to the bundle
		// root
		l := b.Config.GetLocation("resources.jobs." + name)
		if l.File == "" {
			// Skip resources that exist only in the deployment state: statemgmt.Load,
			// which runs before this mutator, injects them into the config without a
			// file location.
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
	return nil
}
