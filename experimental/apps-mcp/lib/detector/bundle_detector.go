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

	// Detect all resource types present in the bundle
	hasApps := false
	for _, group := range b.Config.Resources.AllResources() {
		if len(group.Resources) > 0 {
			detected.TargetTypes = append(detected.TargetTypes, group.Description.PluralName)
			if group.Description.PluralName == "apps" {
				hasApps = true
			}
		}
	}

	// Determine if this is an app-only project (only app resources, nothing else).
	// App-only projects get focused app guidance; others get general bundle guidance.
	isAppOnly := hasApps && len(detected.TargetTypes) == 1

	detected.IsAppOnly = isAppOnly

	// Include general "bundle" guidance for all projects except app-only projects
	if !isAppOnly {
		detected.TargetTypes = append(detected.TargetTypes, "bundle")
	}

	return nil
}
