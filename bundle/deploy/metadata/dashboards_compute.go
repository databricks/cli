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
	resourceMetadata.Register("dashboards", computeDashboards)
}

func computeDashboards(_ context.Context, b *bundle.Bundle, _ resourcestate.ExportedResourcesMap) diag.Diagnostics {
	// Set dashboard config and file paths in metadata
	dashboardsMetadata := make(map[string]*metadata.DashboardResource)
	for name, dashboard := range b.Config.Resources.Dashboards {
		// Compute config file path the dashboard is defined in, relative to the bundle
		// root
		l := b.Config.GetLocation("resources.dashboards." + name)
		if l.File == "" {
			// Skip resources that exist only in the deployment state: statemgmt.Load,
			// which runs before this mutator, injects them into the config without a
			// file location.
			continue
		}

		relativePath, err := filepath.Rel(b.BundleRootPath, l.File)
		if err != nil {
			return diag.Errorf("failed to compute relative path for dashboard %s: %v", name, err)
		}
		// Metadata for the dashboard
		dashboardsMetadata[name] = &metadata.DashboardResource{
			ID:           dashboard.ID,
			RelativePath: filepath.ToSlash(relativePath),
			FilePath:     dashboard.FilePath,
		}
	}
	b.Metadata.Config.Resources.Dashboards = dashboardsMetadata
	return nil
}
