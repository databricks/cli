package detector

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/libs/logdiag"
)

// BundleDetector detects Databricks bundle configuration.
type BundleDetector struct{}

// Detect loads databricks.yml with all includes and extracts target types.
func (d *BundleDetector) Detect(ctx context.Context, workDir string, detected *DetectedContext) error {
	bundlePath := filepath.Join(workDir, "databricks.yml")
	if _, err := os.Stat(bundlePath); err != nil {
		// no bundle file - not an error, just not a bundle project
		return nil
	}

	// use full bundle loading to get all resources including from includes
	if !logdiag.IsSetup(ctx) {
		ctx = logdiag.InitContext(ctx)
	}
	b, err := bundle.Load(ctx, workDir)
	if err != nil || b == nil {
		return nil
	}

	phases.Load(ctx, b)
	if logdiag.HasError(ctx) {
		return nil
	}

	detected.InProject = true
	detected.BundleInfo = &BundleInfo{
		Name:    b.Config.Bundle.Name,
		RootDir: workDir,
	}

	// extract target types from fully loaded resources
	hasApps := len(b.Config.Resources.Apps) > 0
	hasJobs := len(b.Config.Resources.Jobs) > 0
	hasPipelines := len(b.Config.Resources.Pipelines) > 0

	if hasApps {
		detected.TargetTypes = append(detected.TargetTypes, "apps")
	}
	if hasJobs {
		detected.TargetTypes = append(detected.TargetTypes, "jobs")
	}
	if hasPipelines {
		detected.TargetTypes = append(detected.TargetTypes, "pipelines")
	}

	// Determine if this is an app-only project (only app resources, nothing else).
	// App-only projects get focused app guidance; others get "mixed" guidance.
	isAppOnly := hasApps && !hasJobs && !hasPipelines &&
		len(b.Config.Resources.Clusters) == 0 &&
		len(b.Config.Resources.Dashboards) == 0 &&
		len(b.Config.Resources.Experiments) == 0 &&
		len(b.Config.Resources.ModelServingEndpoints) == 0 &&
		len(b.Config.Resources.RegisteredModels) == 0 &&
		len(b.Config.Resources.Schemas) == 0 &&
		len(b.Config.Resources.QualityMonitors) == 0 &&
		len(b.Config.Resources.Volumes) == 0

	// Include "mixed" guidance for all projects except app-only projects
	if !isAppOnly {
		detected.TargetTypes = append(detected.TargetTypes, "mixed")
	}

	return nil
}
