package appkit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FeatureDependency defines a prompt/input required by a feature.
type FeatureDependency struct {
	ID          string // e.g., "sql_warehouse_id"
	FlagName    string // CLI flag name, e.g., "warehouse" (maps to --warehouse)
	Title       string // e.g., "SQL Warehouse ID"
	Description string // e.g., "Required for executing SQL queries"
	Placeholder string
	Required    bool
}

// Feature represents an optional feature that can be added to an AppKit project.
type Feature struct {
	ID           string
	Name         string
	Description  string
	PluginImport string
	PluginUsage  string
	Dependencies []FeatureDependency
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
				FlagName:    "warehouse",
				Title:       "SQL Warehouse ID",
				Description: "required for SQL queries",
				Required:    true,
			},
		},
	},
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

	featureMap := make(map[string]Feature)
	for _, f := range AvailableFeatures {
		featureMap[f.ID] = f
	}

	var imports []string
	var usages []string

	for _, id := range featureIDs {
		feature, ok := featureMap[id]
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
	featureMap := make(map[string]Feature)
	for _, f := range AvailableFeatures {
		featureMap[f.ID] = f
	}

	seen := make(map[string]bool)
	var deps []FeatureDependency

	for _, id := range featureIDs {
		feature, ok := featureMap[id]
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
