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
					{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "warehouse", Description: "SQL Warehouse for queries"},
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
					{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "warehouse", Permission: "CAN_USE"},
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
					{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "warehouse"},
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
					{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "warehouse"},
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
					{
						Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "warehouse",
						Fields: map[string]manifest.ResourceField{
							"id": {Env: "DATABRICKS_WAREHOUSE_ID"},
						},
					},
				},
			},
		},
	}

	cfg := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"warehouse.id": "abc123"},
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
					{
						Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "warehouse",
						Fields: map[string]manifest.ResourceField{
							"id": {Env: "DATABRICKS_WAREHOUSE_ID"},
						},
					},
				},
			},
		},
	}

	result := generator.GenerateDotEnvExample(plugins)
	assert.Equal(t, "DATABRICKS_WAREHOUSE_ID=your_warehouse_id", result)
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
					{Type: "unknown_type", Alias: "Unknown", ResourceKey: "foo"},
				},
			},
		},
	}

	cfg := generator.Config{
		ProjectName: "test-app",
	}

	// Unknown resource types produce no resource block
	result := generator.GenerateBundleResources(plugins, cfg)
	assert.Empty(t, result)

	// Variables are still generated
	result = generator.GenerateBundleVariables(plugins, cfg)
	assert.Contains(t, result, "  foo_id:")
}

func TestGenerateEmptyResourceType(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "empty",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "", Alias: "Foo", ResourceKey: "foo"},
				},
			},
		},
	}

	cfg := generator.Config{
		ProjectName: "test-app",
	}

	// Empty type generates no resource block
	result := generator.GenerateBundleResources(plugins, cfg)
	assert.Empty(t, result)
}

func TestGenerateBundleResourcesDatabaseType(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "caching",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "sql-warehouse", Permission: "CAN_USE"},
					{Type: "database", Alias: "Database", ResourceKey: "database", Permission: "CAN_CONNECT_AND_CREATE"},
				},
			},
		},
	}

	cfg := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"sql-warehouse": "wh123", "database": "some-id"},
	}

	result := generator.GenerateBundleResources(plugins, cfg)
	assert.Contains(t, result, "- name: sql-warehouse")
	assert.Contains(t, result, "sql_warehouse:")
	assert.Contains(t, result, "id: ${var.sql_warehouse_id}")
	assert.Contains(t, result, "- name: database")
	assert.Contains(t, result, "database:")
	assert.Contains(t, result, "instance_name: ${var.database_instance_name}")
	assert.Contains(t, result, "database_name: ${var.database_database_name}")
	assert.Contains(t, result, "permission: CAN_CONNECT_AND_CREATE")
}

func TestGenerateBundleResourcesDefaultPermissions(t *testing.T) {
	tests := []struct {
		resourceType       string
		expectedPermission string
	}{
		{"sql_warehouse", "CAN_USE"},
		{"job", "CAN_MANAGE_RUN"},
		{"serving_endpoint", "CAN_QUERY"},
		{"secret", "READ"},
		{"experiment", "CAN_READ"},
		{"database", "CAN_CONNECT_AND_CREATE"},
		{"volume", "READ_VOLUME"},
		{"uc_function", "EXECUTE"},
		{"uc_connection", "USE_CONNECTION"},
		{"genie_space", "CAN_VIEW"},
		{"vector_search_index", "CAN_USE"},
		// TODO: uncomment when bundles support app as an app resource type.
		// {"app", "CAN_USE"},
	}

	for _, tt := range tests {
		t.Run(tt.resourceType, func(t *testing.T) {
			plugins := []manifest.Plugin{
				{
					Name: "test",
					Resources: manifest.Resources{
						Required: []manifest.Resource{
							{Type: tt.resourceType, Alias: "Resource", ResourceKey: "res"},
						},
					},
				},
			}
			cfg := generator.Config{ResourceValues: map[string]string{"res": "id1"}}
			result := generator.GenerateBundleResources(plugins, cfg)
			assert.Contains(t, result, "permission: "+tt.expectedPermission)
		})
	}
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
	whFields := map[string]manifest.ResourceField{"id": {Env: "DATABRICKS_WAREHOUSE_ID"}}
	secFields := map[string]manifest.ResourceField{"id": {Env: "SECONDARY_WAREHOUSE_ID"}}

	plugins := []manifest.Plugin{
		{
			Name: "analytics",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "warehouse", Description: "Main warehouse", Fields: whFields},
				},
				Optional: []manifest.Resource{
					{Type: "sql_warehouse", Alias: "Secondary Warehouse", ResourceKey: "secondary_warehouse", Description: "Secondary warehouse", Fields: secFields},
				},
			},
		},
	}

	// Config with only required resource
	cfgRequiredOnly := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"warehouse.id": "wh123"},
	}

	// Config with both required and optional resources
	cfgWithOptional := generator.Config{
		ProjectName:    "test-app",
		ResourceValues: map[string]string{"warehouse.id": "wh123", "secondary_warehouse.id": "wh456"},
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

func TestGenerateResourceYAMLAllTypes(t *testing.T) {
	tests := []struct {
		name             string
		resource         manifest.Resource
		expectContains   []string
		expectNotContain []string
	}{
		{
			name:     "sql_warehouse uses id field",
			resource: manifest.Resource{Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "wh", Permission: "CAN_USE"},
			expectContains: []string{
				"- name: wh",
				"sql_warehouse:",
				"id: ${var.wh_id}",
				"permission: CAN_USE",
			},
		},
		{
			name:     "job uses id field",
			resource: manifest.Resource{Type: "job", Alias: "Job", ResourceKey: "myjob", Permission: "CAN_MANAGE_RUN"},
			expectContains: []string{
				"- name: myjob",
				"job:",
				"id: ${var.myjob_id}",
				"permission: CAN_MANAGE_RUN",
			},
		},
		{
			name:     "serving_endpoint uses name field",
			resource: manifest.Resource{Type: "serving_endpoint", Alias: "Model Endpoint", ResourceKey: "endpoint", Permission: "CAN_QUERY"},
			expectContains: []string{
				"- name: endpoint",
				"serving_endpoint:",
				"name: ${var.endpoint_id}",
				"permission: CAN_QUERY",
			},
		},
		{
			name:     "experiment uses experiment_id field",
			resource: manifest.Resource{Type: "experiment", Alias: "Experiment", ResourceKey: "exp", Permission: "CAN_READ"},
			expectContains: []string{
				"- name: exp",
				"experiment:",
				"experiment_id: ${var.exp_id}",
				"permission: CAN_READ",
			},
		},
		{
			name:     "secret uses scope and key fields",
			resource: manifest.Resource{Type: "secret", Alias: "Secret", ResourceKey: "creds", Permission: "READ"},
			expectContains: []string{
				"- name: creds",
				"secret:",
				"scope: ${var.creds_scope}",
				"key: ${var.creds_key}",
				"permission: READ",
			},
			expectNotContain: []string{"id:"},
		},
		{
			name:     "database uses instance_name and database_name fields",
			resource: manifest.Resource{Type: "database", Alias: "Database", ResourceKey: "cache", Permission: "CAN_CONNECT_AND_CREATE"},
			expectContains: []string{
				"- name: cache",
				"database:",
				"instance_name: ${var.cache_instance_name}",
				"database_name: ${var.cache_database_name}",
				"permission: CAN_CONNECT_AND_CREATE",
			},
			expectNotContain: []string{"id:"},
		},
		{
			name:     "genie_space uses name and space_id fields",
			resource: manifest.Resource{Type: "genie_space", Alias: "Genie Space", ResourceKey: "genie-space", Permission: "CAN_VIEW"},
			expectContains: []string{
				"- name: genie-space",
				"genie_space:",
				"name: Genie Space",
				"space_id: ${var.genie_space_space_id}",
				"permission: CAN_VIEW",
			},
		},
		{
			name:     "volume maps to uc_securable",
			resource: manifest.Resource{Type: "volume", Alias: "UC Volume", ResourceKey: "vol", Permission: "READ_VOLUME"},
			expectContains: []string{
				"- name: vol",
				"uc_securable:",
				"securable_full_name: ${var.vol_id}",
				"securable_type: VOLUME",
				"permission: READ_VOLUME",
			},
			expectNotContain: []string{"volume:"},
		},
		{
			name:     "uc_function maps to uc_securable FUNCTION",
			resource: manifest.Resource{Type: "uc_function", Alias: "UC Function", ResourceKey: "func", Permission: "EXECUTE"},
			expectContains: []string{
				"- name: func",
				"uc_securable:",
				"securable_full_name: ${var.func_id}",
				"securable_type: FUNCTION",
				"permission: EXECUTE",
			},
		},
		{
			name:     "uc_connection maps to uc_securable CONNECTION",
			resource: manifest.Resource{Type: "uc_connection", Alias: "UC Connection", ResourceKey: "conn", Permission: "USE_CONNECTION"},
			expectContains: []string{
				"- name: conn",
				"uc_securable:",
				"securable_full_name: ${var.conn_id}",
				"securable_type: CONNECTION",
				"permission: USE_CONNECTION",
			},
		},
		{
			name:     "vector_search_index uses id field",
			resource: manifest.Resource{Type: "vector_search_index", Alias: "Vector Search Index", ResourceKey: "vector-search-index", Permission: "CAN_USE"},
			expectContains: []string{
				"- name: vector-search-index",
				"vector_search_index:",
				"id: ${var.vector_search_index_id}",
				"permission: CAN_USE",
			},
		},
		// TODO: uncomment when bundles support app as an app resource type.
		// {
		// 	name:     "app uses name field",
		// 	resource: manifest.Resource{Type: "app", Alias: "Databricks App", ResourceKey: "app", Permission: "CAN_USE"},
		// 	expectContains: []string{
		// 		"- name: app",
		// 		"app:",
		// 		"name: ${var.app_id}",
		// 		"permission: CAN_USE",
		// 	},
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plugins := []manifest.Plugin{
				{Name: "test", Resources: manifest.Resources{Required: []manifest.Resource{tt.resource}}},
			}
			cfg := generator.Config{ResourceValues: map[string]string{tt.resource.ResourceKey: "val"}}
			result := generator.GenerateBundleResources(plugins, cfg)
			for _, s := range tt.expectContains {
				assert.Contains(t, result, s)
			}
			for _, s := range tt.expectNotContain {
				assert.NotContains(t, result, s)
			}
		})
	}
}

func TestGenerateMultiFieldVariables(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "test",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{
						Type: "database", Alias: "Database", ResourceKey: "database", Description: "App cache",
						Fields: map[string]manifest.ResourceField{
							"instance_name": {Env: "DB_INSTANCE"},
							"database_name": {Env: "DB_NAME"},
						},
					},
					{
						Type: "secret", Alias: "Secret", ResourceKey: "secret", Description: "Credentials",
						Fields: map[string]manifest.ResourceField{
							"scope": {Env: "SECRET_SCOPE"},
							"key":   {Env: "SECRET_KEY"},
						},
					},
					{
						Type: "genie_space", Alias: "Genie Space", ResourceKey: "genie-space", Description: "AI assistant",
						Fields: map[string]manifest.ResourceField{
							"space_id": {Env: "GENIE_SPACE_ID"},
						},
					},
					{
						Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "sql-warehouse", Description: "Warehouse",
						Fields: map[string]manifest.ResourceField{
							"id": {Env: "WH_ID"},
						},
					},
				},
			},
		},
	}
	cfg := generator.Config{ResourceValues: map[string]string{
		"database.instance_name": "val", "database.database_name": "val",
		"secret.scope": "val", "secret.key": "val",
		"genie-space.space_id": "val", "sql-warehouse.id": "val",
	}}

	vars := generator.GenerateBundleVariables(plugins, cfg)
	// database produces two variables
	assert.Contains(t, vars, "database_instance_name:")
	assert.Contains(t, vars, "database_database_name:")
	assert.NotContains(t, vars, "database_id:")

	// secret produces two variables
	assert.Contains(t, vars, "secret_scope:")
	assert.Contains(t, vars, "secret_key:")
	assert.NotContains(t, vars, "secret_id:")

	// genie_space produces one variable (space_id) using VarPrefix
	assert.Contains(t, vars, "genie_space_space_id:")
	assert.NotContains(t, vars, "genie_space_id:")

	// sql_warehouse produces one variable
	assert.Contains(t, vars, "sql_warehouse_id:")
}

func TestGenerateTargetVariablesMultiField(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "test",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{
						Type: "database", Alias: "Database", ResourceKey: "database",
						Fields: map[string]manifest.ResourceField{
							"instance_name": {Env: "DB_INSTANCE"},
							"database_name": {Env: "DB_NAME"},
						},
					},
					{
						Type: "secret", Alias: "Secret", ResourceKey: "secret",
						Fields: map[string]manifest.ResourceField{
							"scope": {Env: "SECRET_SCOPE"},
							"key":   {Env: "SECRET_KEY"},
						},
					},
				},
			},
		},
	}
	cfg := generator.Config{ResourceValues: map[string]string{
		"database.instance_name": "my-instance",
		"database.database_name": "my-db",
		"secret.scope":           "my-scope",
		"secret.key":             "my-key",
	}}

	result := generator.GenerateTargetVariables(plugins, cfg)
	assert.Contains(t, result, "database_instance_name: my-instance")
	assert.Contains(t, result, "database_database_name: my-db")
	assert.Contains(t, result, "secret_scope: my-scope")
	assert.Contains(t, result, "secret_key: my-key")
}

func TestGenerateWithExplicitFields(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "caching",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{
						Type: "database", Alias: "Database", ResourceKey: "database", Permission: "CAN_CONNECT_AND_CREATE",
						Fields: map[string]manifest.ResourceField{
							"instance_name": {Env: "DB_INSTANCE", Description: "Lakebase instance"},
							"database_name": {Env: "DB_NAME", Description: "Database name"},
						},
					},
				},
			},
		},
	}
	cfg := generator.Config{ResourceValues: map[string]string{
		"database.instance_name": "my-inst",
		"database.database_name": "my-db",
	}}

	// Variables use Fields descriptions
	vars := generator.GenerateBundleVariables(plugins, cfg)
	assert.Contains(t, vars, "database_database_name:")
	assert.Contains(t, vars, "    description: Database name")
	assert.Contains(t, vars, "database_instance_name:")
	assert.Contains(t, vars, "    description: Lakebase instance")

	// Target vars use composite keys
	target := generator.GenerateTargetVariables(plugins, cfg)
	assert.Contains(t, target, "database_instance_name: my-inst")
	assert.Contains(t, target, "database_database_name: my-db")

	// .env uses field-level env vars
	env := generator.GenerateDotEnv(plugins, cfg)
	assert.Contains(t, env, "DB_NAME=my-db")
	assert.Contains(t, env, "DB_INSTANCE=my-inst")

	// .env.example uses field-level placeholders
	example := generator.GenerateDotEnvExample(plugins)
	assert.Contains(t, example, "DB_NAME=your_database_database_name")
	assert.Contains(t, example, "DB_INSTANCE=your_database_instance_name")
}

func TestGenerateFieldsDotEnvSecret(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "auth",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{
						Type: "secret", Alias: "Secret", ResourceKey: "secret", Permission: "READ",
						Fields: map[string]manifest.ResourceField{
							"scope": {Env: "SECRET_SCOPE", Description: "Scope name"},
							"key":   {Env: "SECRET_KEY", Description: "Key name"},
						},
					},
				},
			},
		},
	}
	cfg := generator.Config{ResourceValues: map[string]string{
		"secret.scope": "my-scope",
		"secret.key":   "my-key",
	}}

	env := generator.GenerateDotEnv(plugins, cfg)
	assert.Contains(t, env, "SECRET_KEY=my-key")
	assert.Contains(t, env, "SECRET_SCOPE=my-scope")
}

func TestGenerateOptionalMultiFieldResource(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "test",
			Resources: manifest.Resources{
				Optional: []manifest.Resource{
					{
						Type: "database", Alias: "Database", ResourceKey: "database", Permission: "CAN_CONNECT_AND_CREATE",
						Fields: map[string]manifest.ResourceField{
							"instance_name": {Env: "DB_INSTANCE"},
							"database_name": {Env: "DB_NAME"},
						},
					},
				},
			},
		},
	}

	// No values → optional resource is excluded
	cfgEmpty := generator.Config{ResourceValues: map[string]string{}}
	assert.Empty(t, generator.GenerateBundleVariables(plugins, cfgEmpty))
	assert.Empty(t, generator.GenerateBundleResources(plugins, cfgEmpty))
	assert.Empty(t, generator.GenerateTargetVariables(plugins, cfgEmpty))
	assert.Empty(t, generator.GenerateDotEnv(plugins, cfgEmpty))

	// With values → optional resource is included
	cfgFilled := generator.Config{ResourceValues: map[string]string{
		"database.instance_name": "inst",
		"database.database_name": "db",
	}}
	assert.Contains(t, generator.GenerateBundleVariables(plugins, cfgFilled), "database_instance_name:")
	assert.Contains(t, generator.GenerateBundleResources(plugins, cfgFilled), "database:")
	assert.Contains(t, generator.GenerateTargetVariables(plugins, cfgFilled), "database_instance_name: inst")
	assert.Contains(t, generator.GenerateDotEnv(plugins, cfgFilled), "DB_INSTANCE=inst")
}

func TestGenerateDotEnvExampleWithOptional(t *testing.T) {
	plugins := []manifest.Plugin{
		{
			Name: "analytics",
			Resources: manifest.Resources{
				Required: []manifest.Resource{
					{
						Type: "sql_warehouse", Alias: "SQL Warehouse", ResourceKey: "warehouse",
						Fields: map[string]manifest.ResourceField{"id": {Env: "DATABRICKS_WAREHOUSE_ID"}},
					},
				},
				Optional: []manifest.Resource{
					{
						Type: "sql_warehouse", Alias: "Secondary Warehouse", ResourceKey: "secondary",
						Fields: map[string]manifest.ResourceField{"id": {Env: "SECONDARY_WAREHOUSE_ID"}},
					},
				},
			},
		},
	}

	result := generator.GenerateDotEnvExample(plugins)
	// Required resources are shown normally
	assert.Contains(t, result, "DATABRICKS_WAREHOUSE_ID=your_warehouse_id")
	// Optional resources are commented out
	assert.Contains(t, result, "# SECONDARY_WAREHOUSE_ID=your_secondary_id")
}
