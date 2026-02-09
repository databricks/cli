package generator_test

import (
	"testing"

	"github.com/databricks/cli/libs/apps/generator"
	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/stretchr/testify/assert"
)

func TestGenerateBundleVariables(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "analytics",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "warehouse", Description: "SQL Warehouse for queries"},
				},
			},
		},
	}

	cfg := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"warehouse": "abc123"},
	}

	result := generator.GenerateBundleVariables(plugins, cfg)
	assert.Contains(t, result, "  warehouse_id:")
	assert.Contains(t, result, "    description: SQL Warehouse for queries")
}

func TestGenerateBundleResources(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "analytics",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "warehouse", Permission: "CAN_USE"},
				},
			},
		},
	}

	cfg := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"warehouse": "abc123"},
	}

	result := generator.GenerateBundleResources(plugins, cfg)
	assert.Contains(t, result, "        - name: warehouse")
	assert.Contains(t, result, "          sql_warehouse:")
	assert.Contains(t, result, "            id: ${var.warehouse_id}")
	assert.Contains(t, result, "            permission: CAN_USE")
}

func TestGenerateBundleResourcesDefaultPermission(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "analytics",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "warehouse"},
				},
			},
		},
	}

	cfg := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"warehouse": "abc123"},
	}

	result := generator.GenerateBundleResources(plugins, cfg)
	assert.Contains(t, result, "            permission: CAN_USE")
}

func TestGenerateTargetVariables(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "analytics",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "warehouse"},
				},
			},
		},
	}

	cfg := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"warehouse": "abc123"},
	}

	result := generator.GenerateTargetVariables(plugins, cfg)
	assert.Contains(t, result, "      warehouse_id: abc123")
}

func TestGenerateDotEnv(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "analytics",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "warehouse", Env: "DATABRICKS_WAREHOUSE_ID"},
				},
			},
		},
	}

	cfg := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"warehouse": "abc123"},
	}

	result := generator.GenerateDotEnv(plugins, cfg)
	assert.Equal(t, "DATABRICKS_WAREHOUSE_ID=abc123", result)
}

func TestGenerateDotEnvExample(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "analytics",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "warehouse", Env: "DATABRICKS_WAREHOUSE_ID"},
				},
			},
		},
	}

	result := generator.GenerateDotEnvExample(plugins)
	assert.Equal(t, "DATABRICKS_WAREHOUSE_ID=your_warehouse", result)
}

func TestGenerateEmptyPlugins(t *testing.T) {
	var plugins []manifest.Plugin
	cfg := generator.Config{
		ProjectName: "test-app",
	}

	assert.Empty(t, generator.GenerateBundleVariables(plugins, cfg))
	assert.Empty(t, generator.GenerateBundleResources(plugins, cfg))
	assert.Empty(t, generator.GenerateTargetVariables(plugins, cfg))
	assert.Empty(t, generator.GenerateDotEnv(plugins, cfg))
	assert.Empty(t, generator.GenerateDotEnvExample(plugins))
}

func TestGenerateUnknownResourceType(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "unknown",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "unknown_type", Alias: "foo"},
				},
			},
		},
	}

	cfg := generator.Config{
		ProjectName: "test-app",
	}

	// Unknown resource types are skipped for bundle resources
	result := generator.GenerateBundleResources(plugins, cfg)
	assert.Empty(t, result)

	// But variables are still generated
	result = generator.GenerateBundleVariables(plugins, cfg)
	assert.Contains(t, result, "  foo_id:")
}

func TestGetSelectedPlugins(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"analytics": {Name: "analytics", DisplayName: "Analytics"},
			"server":    {Name: "server", DisplayName: "Server"},
			"auth":      {Name: "auth", DisplayName: "Auth"},
		},
	}

	selected := generator.GetSelectedPlugins(m, []string{"analytics", "auth"})
	assert.Len(t, selected, 2)

	names := make([]string, len(selected))
	for i, p := range selected {
		names[i] = p.Name
	}
	assert.Contains(t, names, "analytics")
	assert.Contains(t, names, "auth")
}

func TestGenerateWithOptionalResources(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "analytics",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "warehouse", Description: "Main warehouse"},
				},
				Optional: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "secondary_warehouse", Description: "Secondary warehouse", Env: "SECONDARY_WAREHOUSE_ID"},
				},
			},
		},
	}

	// Config with only required resource
	cfgRequiredOnly := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"warehouse": "wh123"},
	}

	// Config with both required and optional resources
	cfgWithOptional := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"warehouse": "wh123", "secondary_warehouse": "wh456"},
	}

	// Test bundle variables - required only
	result := generator.GenerateBundleVariables(plugins, cfgRequiredOnly)
	assert.Contains(t, result, "  warehouse_id:")
	assert.NotContains(t, result, "secondary_warehouse_id")

	// Test bundle variables - with optional
	result = generator.GenerateBundleVariables(plugins, cfgWithOptional)
	assert.Contains(t, result, "  warehouse_id:")
	assert.Contains(t, result, "  secondary_warehouse_id:")

	// Test bundle resources - required only
	result = generator.GenerateBundleResources(plugins, cfgRequiredOnly)
	assert.Contains(t, result, "- name: warehouse")
	assert.NotContains(t, result, "secondary_warehouse")

	// Test bundle resources - with optional
	result = generator.GenerateBundleResources(plugins, cfgWithOptional)
	assert.Contains(t, result, "- name: warehouse")
	assert.Contains(t, result, "- name: secondary_warehouse")

	// Test target variables - required only
	result = generator.GenerateTargetVariables(plugins, cfgRequiredOnly)
	assert.Contains(t, result, "      warehouse_id: wh123")
	assert.NotContains(t, result, "secondary_warehouse_id")

	// Test target variables - with optional
	result = generator.GenerateTargetVariables(plugins, cfgWithOptional)
	assert.Contains(t, result, "      warehouse_id: wh123")
	assert.Contains(t, result, "      secondary_warehouse_id: wh456")

	// Test .env - required only
	result = generator.GenerateDotEnv(plugins, cfgRequiredOnly)
	assert.NotContains(t, result, "SECONDARY_WAREHOUSE_ID")

	// Test .env - with optional
	result = generator.GenerateDotEnv(plugins, cfgWithOptional)
	assert.Contains(t, result, "SECONDARY_WAREHOUSE_ID=wh456")
}

func TestGenerateDotEnvExampleWithOptional(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "analytics",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "warehouse", Env: "DATABRICKS_WAREHOUSE_ID"},
				},
				Optional: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "secondary", Env: "SECONDARY_WAREHOUSE_ID"},
				},
			},
		},
	}

	result := generator.GenerateDotEnvExample(plugins)
	// Required resources are shown normally
	assert.Contains(t, result, "DATABRICKS_WAREHOUSE_ID=your_warehouse")
	// Optional resources are commented out
	assert.Contains(t, result, "# SECONDARY_WAREHOUSE_ID=your_secondary")
}
