package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

const ManifestFileName = "appkit.plugins.json"

// Resource defines a Databricks resource required or optional for a plugin.
type Resource struct {
	Type        string `json:"type"`        // e.g., "sql_warehouse"
	Alias       string `json:"alias"`       // e.g., "warehouse"
	Description string `json:"description"` // e.g., "SQL Warehouse for executing analytics queries"
	Permission  string `json:"permission"`  // e.g., "CAN_USE"
	Env         string `json:"env"`         // e.g., "DATABRICKS_WAREHOUSE_ID"
}

// Resources defines the required and optional resources for a plugin.
type Resources struct {
	Required []Resource `json:"required"`
	Optional []Resource `json:"optional"`
}

// Plugin represents a plugin defined in the manifest.
type Plugin struct {
	Name        string    `json:"name"`
	DisplayName string    `json:"displayName"`
	Description string    `json:"description"`
	Package     string    `json:"package"`
	Resources   Resources `json:"resources"`
}

// Manifest represents the appkit.plugins.json file structure.
type Manifest struct {
	Schema  string            `json:"$schema"`
	Version string            `json:"version"`
	Plugins map[string]Plugin `json:"plugins"`
}

// Load reads and parses the appkit.plugins.json manifest from the template directory.
func Load(templateDir string) (*Manifest, error) {
	path := filepath.Join(templateDir, ManifestFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("manifest file not found: %s", path)
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &m, nil
}

// HasManifest checks if the template directory contains an appkit.plugins.json file.
func HasManifest(templateDir string) bool {
	path := filepath.Join(templateDir, ManifestFileName)
	_, err := os.Stat(path)
	return err == nil
}

// GetPlugins returns all plugins from the manifest sorted by name.
// The plugin name is taken from the map key if not specified in the plugin object.
func (m *Manifest) GetPlugins() []Plugin {
	plugins := make([]Plugin, 0, len(m.Plugins))
	for name, p := range m.Plugins {
		if p.Name == "" {
			p.Name = name
		}
		plugins = append(plugins, p)
	}
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Name < plugins[j].Name
	})
	return plugins
}

// GetSelectablePlugins returns plugins that have resources (can be selected/configured).
// Plugins without resources (like a base server) are considered always-on.
func (m *Manifest) GetSelectablePlugins() []Plugin {
	var selectable []Plugin
	for _, p := range m.GetPlugins() {
		if len(p.Resources.Required) > 0 || len(p.Resources.Optional) > 0 {
			selectable = append(selectable, p)
		}
	}
	return selectable
}

// GetPluginByName returns a plugin by its name, or nil if not found.
func (m *Manifest) GetPluginByName(name string) *Plugin {
	if p, ok := m.Plugins[name]; ok {
		return &p
	}
	return nil
}

// GetPluginNames returns a list of all plugin names.
func (m *Manifest) GetPluginNames() []string {
	names := make([]string, 0, len(m.Plugins))
	for name := range m.Plugins {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ValidatePluginNames checks that all provided plugin names exist in the manifest.
func (m *Manifest) ValidatePluginNames(names []string) error {
	for _, name := range names {
		if _, ok := m.Plugins[name]; !ok {
			return fmt.Errorf("unknown plugin: %q; available: %v", name, m.GetPluginNames())
		}
	}
	return nil
}

// CollectResources returns all required resources for the given plugin names.
func (m *Manifest) CollectResources(pluginNames []string) []Resource {
	seen := make(map[string]bool)
	var resources []Resource

	for _, name := range pluginNames {
		plugin := m.GetPluginByName(name)
		if plugin == nil {
			continue
		}
		for _, r := range plugin.Resources.Required {
			key := r.Type + ":" + r.Alias
			if !seen[key] {
				seen[key] = true
				resources = append(resources, r)
			}
		}
	}

	return resources
}

// CollectOptionalResources returns all optional resources for the given plugin names.
func (m *Manifest) CollectOptionalResources(pluginNames []string) []Resource {
	seen := make(map[string]bool)
	var resources []Resource

	for _, name := range pluginNames {
		plugin := m.GetPluginByName(name)
		if plugin == nil {
			continue
		}
		for _, r := range plugin.Resources.Optional {
			key := r.Type + ":" + r.Alias
			if !seen[key] {
				seen[key] = true
				resources = append(resources, r)
			}
		}
	}

	return resources
}
