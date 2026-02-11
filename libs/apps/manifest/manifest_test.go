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
							"alias": "SQL Warehouse",
							"resourceKey": "sql-warehouse",
							"description": "SQL Warehouse",
							"permission": "CAN_USE",
							"fields": {
								"id": {"env": "DATABRICKS_WAREHOUSE_ID", "description": "SQL Warehouse ID"}
							}
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
				"requiredByTemplate": true,
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
	assert.True(t, m.Plugins["server"].RequiredByTemplate)
	assert.False(t, m.Plugins["analytics"].RequiredByTemplate)
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
				Name:               "server",
				RequiredByTemplate: true,
			},
			"analytics": {
				Name: "analytics",
				Resources: manifest.Resources{
					Required: []manifest.Resource{
						{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "sql-warehouse"},
					},
				},
			},
			"optional-plugin": {
				Name: "optional-plugin",
			},
		},
	}

	selectable := m.GetSelectablePlugins()
	require.Len(t, selectable, 2)
	assert.Equal(t, "analytics", selectable[0].Name)
	assert.Equal(t, "optional-plugin", selectable[1].Name)
}

func TestGetMandatoryPlugins(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"server": {
				Name:               "server",
				RequiredByTemplate: true,
			},
			"analytics": {
				Name: "analytics",
			},
			"core": {
				Name:               "core",
				RequiredByTemplate: true,
				Resources: manifest.Resources{
					Required: []manifest.Resource{
						{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "sql-warehouse"},
					},
				},
			},
		},
	}

	mandatory := m.GetMandatoryPlugins()
	require.Len(t, mandatory, 2)
	assert.Equal(t, "core", mandatory[0].Name)
	assert.Equal(t, "server", mandatory[1].Name)

	names := m.GetMandatoryPluginNames()
	assert.Equal(t, []string{"core", "server"}, names)
}

func TestGetMandatoryPluginsEmpty(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"analytics": {Name: "analytics"},
		},
	}

	mandatory := m.GetMandatoryPlugins()
	assert.Empty(t, mandatory)
	assert.Empty(t, m.GetMandatoryPluginNames())
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
						{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "sql-warehouse"},
					},
				},
			},
			"genie": {
				Name: "genie",
				Resources: manifest.Resources{
					Required: []manifest.Resource{
						{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "sql-warehouse"},
						{Type: "genie_space", Alias: "Genie Space", ResourceKey: "genie-space"},
					},
				},
			},
		},
	}

	resources := m.CollectResources([]string{"analytics"})
	require.Len(t, resources, 1)
	assert.Equal(t, "sql_warehouse", resources[0].Type)

	// Collect from both - warehouse should be deduplicated by resource_key
	resources = m.CollectResources([]string{"analytics", "genie"})
	require.Len(t, resources, 2)
}

func TestResourceFields(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, manifest.ManifestFileName)

	content := `{
		"version": "1.0",
		"plugins": {
			"caching": {
				"name": "caching",
				"displayName": "Caching",
				"description": "DB caching",
				"package": "@databricks/appkit",
				"resources": {
					"required": [
						{
							"type": "database",
							"alias": "Database",
							"resourceKey": "database",
							"description": "Cache database",
							"permission": "CAN_CONNECT_AND_CREATE",
							"fields": {
								"instance_name": {"env": "DB_INSTANCE", "description": "Lakebase instance"},
								"database_name": {"env": "DB_NAME", "description": "Database name"}
							}
						}
					],
					"optional": []
				}
			}
		}
	}`

	err := os.WriteFile(manifestPath, []byte(content), 0o644)
	require.NoError(t, err)

	m, err := manifest.Load(dir)
	require.NoError(t, err)

	p := m.GetPluginByName("caching")
	require.NotNil(t, p)
	require.Len(t, p.Resources.Required, 1)

	r := p.Resources.Required[0]
	assert.True(t, r.HasFields())
	assert.Len(t, r.Fields, 2)
	assert.Equal(t, "DB_INSTANCE", r.Fields["instance_name"].Env)
	assert.Equal(t, "DB_NAME", r.Fields["database_name"].Env)
	assert.Equal(t, []string{"database_name", "instance_name"}, r.FieldNames())
}

func TestResourceHasFieldsFalse(t *testing.T) {
	r := manifest.Resource{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "sql-warehouse"}
	assert.False(t, r.HasFields())
	assert.Empty(t, r.FieldNames())
}

func TestResourceKey(t *testing.T) {
	r := manifest.Resource{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "sql-warehouse"}
	assert.Equal(t, "sql-warehouse", r.Key())
	assert.Equal(t, "sql_warehouse", r.VarPrefix())
}

func TestCollectOptionalResources(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"analytics": {
				Name: "analytics",
				Resources: manifest.Resources{
					Optional: []manifest.Resource{
						{Type: "catalog", Alias: "Default Catalog", ResourceKey: "default-catalog"},
					},
				},
			},
		},
	}

	resources := m.CollectOptionalResources([]string{"analytics"})
	require.Len(t, resources, 1)
	assert.Equal(t, "catalog", resources[0].Type)
}
