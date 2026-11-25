package metadata

import (
	"context"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/metadata"
	"github.com/databricks/cli/bundle/statemgmt/resourcestate"
	"github.com/databricks/cli/libs/diag"
)

func init() {
	resourceMetadata.Register("pipelines", computePipelines)
}

func computePipelines(_ context.Context, b *bundle.Bundle, _ resourcestate.ExportedResourcesMap) diag.Diagnostics {
	// Set pipeline config paths in metadata
	pipelinesMetadata := make(map[string]*metadata.Resource)
	for name, pipeline := range b.Config.Resources.Pipelines {
		// Compute config file path the pipeline is defined in, relative to the bundle
		// root
		l := b.Config.GetLocation("resources.pipelines." + name)
		if l.File == "" {
			// Skip resources that exist only in the deployment state: statemgmt.Load,
			// which runs before this mutator, injects them into the config without a
			// file location.
			continue
		}

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
	return nil
}
