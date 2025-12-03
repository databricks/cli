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
	ctx = logdiag.InitContext(ctx)
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
	if len(b.Config.Resources.Apps) > 0 {
		detected.TargetTypes = append(detected.TargetTypes, "apps")
	}
	if len(b.Config.Resources.Jobs) > 0 {
		detected.TargetTypes = append(detected.TargetTypes, "jobs")
	}
	if len(b.Config.Resources.Pipelines) > 0 {
		detected.TargetTypes = append(detected.TargetTypes, "pipelines")
	}

	return nil
}
