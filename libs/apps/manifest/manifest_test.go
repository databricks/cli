package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, manifest.ManifestFileName)

	content := `{
		"$schema": "https://databricks.github.io/appkit/schemas/template-plugins.schema.json",
		"version": "1.0",
		"plugins": {
			"analytics": {
				"name": "analytics",
				"displayName": "Analytics Plugin",
				"description": "SQL query execution",
				"package": "@databricks/appkit",
				"resources": {
					"required": [
						{
							"type": "sql_warehouse",
							"alias": "warehouse",
							"description": "SQL Warehouse",
							"permission": "CAN_USE",
							"env": "DATABRICKS_WAREHOUSE_ID"
						}
					],
					"optional": []
				}
			},
			"server": {
				"name": "server",
				"displayName": "Server Plugin",
				"description": "HTTP server",
				"package": "@databricks/appkit",
				"resources": {
					"required": [],
					"optional": []
				}
			}
		}
	}`

	err := os.WriteFile(manifestPath, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := manifest.Load(dir)
	require.NoError(t, err)
	assert.Equal(t, "1.0", m.Version)
	assert.Len(t, m.Plugins, 2)
}

func TestLoadNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := manifest.Load(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "manifest file not found")
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, manifest.ManifestFileName)

	err := os.WriteFile(manifestPath, []byte("invalid json"), 0o644)
	require.NoError(t, err)

	_, err = manifest.Load(dir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse manifest")
}

func TestHasManifest(t *testing.T) {
	dir := t.TempDir()
	assert.False(t, manifest.HasManifest(dir))

	manifestPath := filepath.Join(dir, manifest.ManifestFileName)
	err := os.WriteFile(manifestPath, []byte("{}"), 0o644)
	require.NoError(t, err)

	assert.True(t, manifest.HasManifest(dir))
}

func TestGetPlugins(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"zebra": {Name: "zebra", DisplayName: "Zebra"},
			"alpha": {Name: "alpha", DisplayName: "Alpha"},
		},
	}

	plugins := m.GetPlugins()
	require.Len(t, plugins, 2)
	assert.Equal(t, "alpha", plugins[0].Name)
	assert.Equal(t, "zebra", plugins[1].Name)
}

func TestGetSelectablePlugins(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"server": {
				Name: "server",
				Resources: manifest.Resources{
					Required: []manifest.Resource{},
					Optional: []manifest.Resource{},
				},
			},
			"analytics": {
				Name: "analytics",
				Resources: manifest.Resources{
					Required: []manifest.Resource{
						{Type: "sql_warehouse", Alias: "warehouse"},
					},
					Optional: []manifest.Resource{},
				},
			},
		},
	}

	selectable := m.GetSelectablePlugins()
	require.Len(t, selectable, 1)
	assert.Equal(t, "analytics", selectable[0].Name)
}

func TestGetPluginByName(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"analytics": {Name: "analytics", DisplayName: "Analytics"},
		},
	}

	p := m.GetPluginByName("analytics")
	require.NotNil(t, p)
	assert.Equal(t, "Analytics", p.DisplayName)

	p = m.GetPluginByName("nonexistent")
	assert.Nil(t, p)
}

func TestGetPluginNames(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"zebra": {Name: "zebra"},
			"alpha": {Name: "alpha"},
		},
	}

	names := m.GetPluginNames()
	require.Len(t, names, 2)
	assert.Equal(t, "alpha", names[0])
	assert.Equal(t, "zebra", names[1])
}

func TestValidatePluginNames(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"analytics": {Name: "analytics"},
			"server":    {Name: "server"},
		},
	}

	err := m.ValidatePluginNames([]string{"analytics"})
	assert.NoError(t, err)

	err = m.ValidatePluginNames([]string{"analytics", "server"})
	assert.NoError(t, err)

	err = m.ValidatePluginNames([]string{"nonexistent"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown plugin")
}

func TestCollectResources(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"analytics": {
				Name: "analytics",
				Resources: manifest.Resources{
					Required: []manifest.Resource{
						{Type: "sql_warehouse", Alias: "warehouse", Env: "DATABRICKS_WAREHOUSE_ID"},
					},
				},
			},
			"genie": {
				Name: "genie",
				Resources: manifest.Resources{
					Required: []manifest.Resource{
						{Type: "sql_warehouse", Alias: "warehouse", Env: "DATABRICKS_WAREHOUSE_ID"},
						{Type: "genie_space", Alias: "genie", Env: "GENIE_SPACE_ID"},
					},
				},
			},
		},
	}

	resources := m.CollectResources([]string{"analytics"})
	require.Len(t, resources, 1)
	assert.Equal(t, "sql_warehouse", resources[0].Type)

	// Collect from both - warehouse should be deduplicated
	resources = m.CollectResources([]string{"analytics", "genie"})
	require.Len(t, resources, 2)
}

func TestCollectOptionalResources(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"analytics": {
				Name: "analytics",
				Resources: manifest.Resources{
					Optional: []manifest.Resource{
						{Type: "catalog", Alias: "default_catalog", Env: "DEFAULT_CATALOG"},
					},
				},
			},
		},
	}

	resources := m.CollectOptionalResources([]string{"analytics"})
	require.Len(t, resources, 1)
	assert.Equal(t, "catalog", resources[0].Type)
}
