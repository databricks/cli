package features

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// FeatureDependency defines a prompt/input required by a feature.
type FeatureDependency struct {
	ID          string // e.g., "sql_warehouse_id"
	FlagName    string // CLI flag name, e.g., "warehouse-id" (maps to --warehouse-id)
	Title       string // e.g., "SQL Warehouse ID"
	Description string // e.g., "Required for executing SQL queries"
	Placeholder string
	Required    bool
}

// FeatureResourceFiles defines paths to YAML fragment files for a feature's resources.
// Paths are relative to the template's features directory (e.g., "analytics/bundle_variables.yml").
type FeatureResourceFiles struct {
	BundleVariables string // Variables section for databricks.yml
	BundleResources string // Resources section for databricks.yml (app resources)
	TargetVariables string // Dev target variables section for databricks.yml
	AppEnv          string // Environment variables for app.yaml
	DotEnv          string // Environment variables for .env (development)
	DotEnvExample   string // Environment variables for .env.example
}

// Feature represents an optional feature that can be added to an AppKit project.
type Feature struct {
	ID            string
	Name          string
	Description   string
	PluginImport  string
	PluginUsage   string
	Dependencies  []FeatureDependency
	ResourceFiles FeatureResourceFiles
}

// AvailableFeatures lists all features that can be selected when creating a project.
var AvailableFeatures = []Feature{
	{
		ID:           "analytics",
		Name:         "Analytics",
		Description:  "SQL analytics with charts and dashboards",
		PluginImport: "analytics",
		PluginUsage:  "analytics()",
		Dependencies: []FeatureDependency{
			{
				ID:          "sql_warehouse_id",
				FlagName:    "warehouse-id",
				Title:       "SQL Warehouse ID",
				Description: "required for SQL queries",
				Required:    true,
			},
		},
		ResourceFiles: FeatureResourceFiles{
			BundleVariables: "analytics/bundle_variables.yml",
			BundleResources: "analytics/bundle_resources.yml",
			TargetVariables: "analytics/target_variables.yml",
			AppEnv:          "analytics/app_env.yml",
			DotEnv:          "analytics/dotenv.yml",
			DotEnvExample:   "analytics/dotenv_example.yml",
		},
	},
}

var featureByID = func() map[string]Feature {
	m := make(map[string]Feature, len(AvailableFeatures))
	for _, f := range AvailableFeatures {
		m[f.ID] = f
	}
	return m
}()

// featureByPluginImport maps plugin import names to features.
var featureByPluginImport = func() map[string]Feature {
	m := make(map[string]Feature, len(AvailableFeatures))
	for _, f := range AvailableFeatures {
		if f.PluginImport != "" {
			m[f.PluginImport] = f
		}
	}
	return m
}()

// pluginPattern matches plugin function calls dynamically built from AvailableFeatures.
// Matches patterns like: analytics(), genie(), oauth(), etc.
var pluginPattern = func() *regexp.Regexp {
	var plugins []string
	for _, f := range AvailableFeatures {
		if f.PluginImport != "" {
			plugins = append(plugins, regexp.QuoteMeta(f.PluginImport))
		}
	}
	if len(plugins) == 0 {
		// Fallback pattern that matches nothing
		return regexp.MustCompile(`$^`)
	}
	// Build pattern: \b(plugin1|plugin2|plugin3)\s*\(
	pattern := `\b(` + strings.Join(plugins, "|") + `)\s*\(`
	return regexp.MustCompile(pattern)
}()

// serverFilePaths lists common locations for the server entry file.
var serverFilePaths = []string{
	"src/server/index.ts",
	"src/server/index.tsx",
	"src/server.ts",
	"server/index.ts",
	"server/server.ts",
	"server.ts",
}

// TODO: We should come to an agreement if we want to do it like this,
// or maybe we should have an appkit.json manifest file in each project.
func DetectPluginsFromServer(templateDir string) ([]string, error) {
	var content []byte

	for _, p := range serverFilePaths {
		fullPath := filepath.Join(templateDir, p)
		data, err := os.ReadFile(fullPath)
		if err == nil {
			content = data
			break
		}
	}

	if content == nil {
		return nil, nil // No server file found
	}

	matches := pluginPattern.FindAllStringSubmatch(string(content), -1)
	seen := make(map[string]bool)
	var plugins []string

	for _, m := range matches {
		plugin := m[1]
		if !seen[plugin] {
			seen[plugin] = true
			plugins = append(plugins, plugin)
		}
	}

	return plugins, nil
}

// GetPluginDependencies returns all dependencies required by the given plugin names.
func GetPluginDependencies(pluginNames []string) []FeatureDependency {
	seen := make(map[string]bool)
	var deps []FeatureDependency

	for _, plugin := range pluginNames {
		feature, ok := featureByPluginImport[plugin]
		if !ok {
			continue
		}
		for _, dep := range feature.Dependencies {
			if !seen[dep.ID] {
				seen[dep.ID] = true
				deps = append(deps, dep)
			}
		}
	}

	return deps
}

// MapPluginsToFeatures maps plugin import names to feature IDs.
// This is used to convert detected plugins (e.g., "analytics") to feature IDs
// so that ApplyFeatures can properly retain feature-specific files.
func MapPluginsToFeatures(pluginNames []string) []string {
	seen := make(map[string]bool)
	var featureIDs []string

	for _, plugin := range pluginNames {
		feature, ok := featureByPluginImport[plugin]
		if ok && !seen[feature.ID] {
			seen[feature.ID] = true
			featureIDs = append(featureIDs, feature.ID)
		}
	}

	return featureIDs
}

// HasFeaturesDirectory checks if the template uses the feature-fragment system.
func HasFeaturesDirectory(templateDir string) bool {
	featuresDir := filepath.Join(templateDir, "features")
	info, err := os.Stat(featuresDir)
	return err == nil && info.IsDir()
}

// ValidateFeatureIDs checks that all provided feature IDs are valid.
// Returns an error if any feature ID is unknown.
func ValidateFeatureIDs(featureIDs []string) error {
	for _, id := range featureIDs {
		if _, ok := featureByID[id]; !ok {
			return fmt.Errorf("unknown feature: %q; available: %s", id, strings.Join(GetFeatureIDs(), ", "))
		}
	}
	return nil
}

// ValidateFeatureDependencies checks that all required dependencies for the given features
// are provided in the flagValues map. Returns an error listing missing required flags.
func ValidateFeatureDependencies(featureIDs []string, flagValues map[string]string) error {
	deps := CollectDependencies(featureIDs)
	var missing []string

	for _, dep := range deps {
		if !dep.Required {
			continue
		}
		value, ok := flagValues[dep.FlagName]
		if !ok || value == "" {
			missing = append(missing, "--"+dep.FlagName)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required flags for selected features: %s", strings.Join(missing, ", "))
	}
	return nil
}

// GetFeatureIDs returns a list of all available feature IDs for help text.
func GetFeatureIDs() []string {
	ids := make([]string, len(AvailableFeatures))
	for i, f := range AvailableFeatures {
		ids[i] = f.ID
	}
	return ids
}

// BuildPluginStrings builds the plugin import and usage strings from selected feature IDs.
// Returns comma-separated imports and newline-separated usages.
func BuildPluginStrings(featureIDs []string) (pluginImport, pluginUsage string) {
	if len(featureIDs) == 0 {
		return "", ""
	}

	var imports []string
	var usages []string

	for _, id := range featureIDs {
		feature, ok := featureByID[id]
		if !ok || feature.PluginImport == "" {
			continue
		}
		imports = append(imports, feature.PluginImport)
		usages = append(usages, feature.PluginUsage)
	}

	if len(imports) == 0 {
		return "", ""
	}

	// Join imports with comma (e.g., "analytics, trpc")
	pluginImport = strings.Join(imports, ", ")

	// Join usages with newline and proper indentation
	pluginUsage = strings.Join(usages, ",\n    ")

	return pluginImport, pluginUsage
}

// ApplyFeatures applies any post-copy modifications for selected features.
// This removes feature-specific directories if the feature is not selected.
func ApplyFeatures(projectDir string, featureIDs []string) error {
	selectedSet := make(map[string]bool)
	for _, id := range featureIDs {
		selectedSet[id] = true
	}

	// Remove analytics-specific files if analytics is not selected
	if !selectedSet["analytics"] {
		queriesDir := filepath.Join(projectDir, "config", "queries")
		if err := os.RemoveAll(queriesDir); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// CollectDependencies returns all unique dependencies required by the selected features.
func CollectDependencies(featureIDs []string) []FeatureDependency {
	seen := make(map[string]bool)
	var deps []FeatureDependency

	for _, id := range featureIDs {
		feature, ok := featureByID[id]
		if !ok {
			continue
		}
		for _, dep := range feature.Dependencies {
			if !seen[dep.ID] {
				seen[dep.ID] = true
				deps = append(deps, dep)
			}
		}
	}

	return deps
}

// CollectResourceFiles returns all resource file paths for the selected features.
func CollectResourceFiles(featureIDs []string) []FeatureResourceFiles {
	var resources []FeatureResourceFiles
	for _, id := range featureIDs {
		feature, ok := featureByID[id]
		if !ok {
			continue
		}
		// Only include if at least one resource file is defined
		rf := feature.ResourceFiles
		if rf.BundleVariables != "" || rf.BundleResources != "" ||
			rf.TargetVariables != "" || rf.AppEnv != "" ||
			rf.DotEnv != "" || rf.DotEnvExample != "" {
			resources = append(resources, rf)
		}
	}

	return resources
}
