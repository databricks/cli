package detector

import (
	"context"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// BundleDetector detects Databricks bundle configuration.
type BundleDetector struct{}

// bundleConfig represents the relevant parts of databricks.yml.
type bundleConfig struct {
	Bundle struct {
		Name string `yaml:"name"`
	} `yaml:"bundle"`
	Resources struct {
		Apps      map[string]any `yaml:"apps"`
		Jobs      map[string]any `yaml:"jobs"`
		Pipelines map[string]any `yaml:"pipelines"`
	} `yaml:"resources"`
}

// Detect parses databricks.yml and extracts target types.
func (d *BundleDetector) Detect(ctx context.Context, workDir string, detected *DetectedContext) error {
	bundlePath := filepath.Join(workDir, "databricks.yml")

	data, err := os.ReadFile(bundlePath)
	if err != nil {
		// no bundle file - not an error, just not a bundle project
		return nil
	}

	var config bundleConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil
	}

	detected.InProject = true
	detected.BundleInfo = &BundleInfo{
		Name:    config.Bundle.Name,
		RootDir: workDir,
	}

	// extract target types from resources
	if len(config.Resources.Apps) > 0 {
		detected.TargetTypes = append(detected.TargetTypes, "apps")
	}
	if len(config.Resources.Jobs) > 0 {
		detected.TargetTypes = append(detected.TargetTypes, "jobs")
	}
	if len(config.Resources.Pipelines) > 0 {
		detected.TargetTypes = append(detected.TargetTypes, "pipelines")
	}

	return nil
}
