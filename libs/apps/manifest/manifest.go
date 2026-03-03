package manifest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const ManifestFileName = "appkit.plugins.json"

// ResourceField describes a single field within a multi-field resource.
// Multi-field resources (e.g., database, secret) need separate env vars and values per field.
type ResourceField struct {
	Env         string `json:"env"`
	Description string `json:"description"`
}

// Resource defines a Databricks resource required or optional for a plugin.
type Resource struct {
	Type        string                   `json:"type"`        // e.g., "sql_warehouse"
	Alias       string                   `json:"alias"`       // display name, e.g., "SQL Warehouse"
	ResourceKey string                   `json:"resourceKey"` // machine key for config/env, e.g., "sql-warehouse"
	Description string                   `json:"description"` // e.g., "SQL Warehouse for executing analytics queries"
	Permission  string                   `json:"permission"`  // e.g., "CAN_USE"
	Fields      map[string]ResourceField `json:"fields"`      // field definitions with env var mappings
}

// Key returns the resource key for machine use (config keys, variable naming).
func (r Resource) Key() string {
	return r.ResourceKey
}

// VarPrefix returns the variable name prefix derived from the resource key.
// Hyphens are replaced with underscores for YAML variable name compatibility.
func (r Resource) VarPrefix() string {
	return strings.ReplaceAll(r.Key(), "-", "_")
}

// HasFields returns true if the resource has explicit field definitions.
func (r Resource) HasFields() bool {
	return len(r.Fields) > 0
}

// FieldNames returns the field names in sorted order for deterministic iteration.
func (r Resource) FieldNames() []string {
	names := make([]string, 0, len(r.Fields))
	for k := range r.Fields {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// Resources defines the required and optional resources for a plugin.
type Resources struct {
	Required []Resource `json:"required"`
	Optional []Resource `json:"optional"`
}

// Plugin represents a plugin defined in the manifest.
type Plugin struct {
	Name               string    `json:"name"`
	DisplayName        string    `json:"displayName"`
	Description        string    `json:"description"`
	Package            string    `json:"package"`
	RequiredByTemplate bool      `json:"requiredByTemplate"`
	Resources          Resources `json:"resources"`
	OnSetupMessage     string    `json:"onSetupMessage"`
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

// GetSelectablePlugins returns plugins the user can choose during init.
// Excludes mandatory plugins (they are always included automatically).
func (m *Manifest) GetSelectablePlugins() []Plugin {
	var selectable []Plugin
	for _, p := range m.GetPlugins() {
		if !p.RequiredByTemplate {
			selectable = append(selectable, p)
		}
	}
	return selectable
}

// GetMandatoryPlugins returns plugins marked as requiredByTemplate.
func (m *Manifest) GetMandatoryPlugins() []Plugin {
	var mandatory []Plugin
	for _, p := range m.GetPlugins() {
		if p.RequiredByTemplate {
			mandatory = append(mandatory, p)
		}
	}
	return mandatory
}

// GetMandatoryPluginNames returns the names of all mandatory plugins.
func (m *Manifest) GetMandatoryPluginNames() []string {
	var names []string
	for _, p := range m.GetMandatoryPlugins() {
		names = append(names, p.Name)
	}
	return names
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
			// TODO: remove skip when bundles support app as an app resource type.
			if r.Type == "app" {
				continue
			}
			key := r.Type + ":" + r.Key()
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
			// TODO: remove skip when bundles support app as an app resource type.
			if r.Type == "app" {
				continue
			}
			key := r.Type + ":" + r.Key()
			if !seen[key] {
				seen[key] = true
				resources = append(resources, r)
			}
		}
	}

	return resources
}
