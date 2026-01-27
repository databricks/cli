package apps

import (
	"bytes"
	"strings"
	"testing"
	"text/template"

	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		// Text files by extension
		{"file.ts", true},
		{"file.tsx", true},
		{"file.js", true},
		{"file.jsx", true},
		{"file.json", true},
		{"file.yaml", true},
		{"file.yml", true},
		{"file.md", true},
		{"file.txt", true},
		{"file.html", true},
		{"file.css", true},
		{"file.scss", true},
		{"file.sql", true},
		{"file.sh", true},
		{"file.py", true},
		{"file.go", true},
		{"file.toml", true},
		{"file.env", true},

		// Text files by name
		{"Makefile", true},
		{"Dockerfile", true},
		{"LICENSE", true},
		{"README", true},
		{".gitignore", true},
		{".env", true},
		{"_gitignore", true},
		{"_env", true},

		// Binary files (should return false)
		{"file.png", false},
		{"file.jpg", false},
		{"file.gif", false},
		{"file.pdf", false},
		{"file.exe", false},
		{"file.bin", false},
		{"file.zip", false},
		{"randomfile", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isTextFile(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubstituteVars(t *testing.T) {
	vars := templateVars{
		ProjectName:    "my-app",
		SQLWarehouseID: "warehouse123",
		AppDescription: "My awesome app",
		Profile:        "default",
		WorkspaceHost:  "https://dbc-123.cloud.databricks.com",
		PluginImport:   "analytics",
		PluginUsage:    "analytics()",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "project name substitution",
			input:    "name: {{.project_name}}",
			expected: "name: my-app",
		},
		{
			name:     "warehouse id substitution",
			input:    "warehouse: {{.sql_warehouse_id}}",
			expected: "warehouse: warehouse123",
		},
		{
			name:     "description substitution",
			input:    "description: {{.app_description}}",
			expected: "description: My awesome app",
		},
		{
			name:     "profile substitution",
			input:    "profile: {{.profile}}",
			expected: "profile: default",
		},
		{
			name:     "workspace host substitution",
			input:    "host: {{workspace_host}}",
			expected: "host: https://dbc-123.cloud.databricks.com",
		},
		{
			name:     "plugin import substitution",
			input:    "import { {{.plugin_import}} } from 'appkit'",
			expected: "import { analytics } from 'appkit'",
		},
		{
			name:     "plugin usage substitution",
			input:    "plugins: [{{.plugin_usage}}]",
			expected: "plugins: [analytics()]",
		},
		{
			name:     "multiple substitutions",
			input:    "{{.project_name}} - {{.app_description}}",
			expected: "my-app - My awesome app",
		},
		{
			name:     "no substitutions needed",
			input:    "plain text without variables",
			expected: "plain text without variables",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteVars(tt.input, vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSubstituteVarsNoPlugins(t *testing.T) {
	// Test plugin cleanup when no plugins are selected
	vars := templateVars{
		ProjectName:    "my-app",
		SQLWarehouseID: "",
		AppDescription: "My app",
		Profile:        "",
		WorkspaceHost:  "",
		PluginImport:   "", // No plugins
		PluginUsage:    "",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes plugin import with comma",
			input:    "import { core, {{.plugin_import}} } from 'appkit'",
			expected: "import { core } from 'appkit'",
		},
		{
			name:     "removes plugin usage line",
			input:    "plugins: [\n    {{.plugin_usage}},\n]",
			expected: "plugins: [\n]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := substituteVars(tt.input, vars)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseDeployAndRunFlags(t *testing.T) {
	tests := []struct {
		name        string
		deploy      bool
		run         string
		wantDeploy  bool
		wantRunMode prompt.RunMode
		wantErr     bool
	}{
		{
			name:        "deploy true, run none",
			deploy:      true,
			run:         "none",
			wantDeploy:  true,
			wantRunMode: prompt.RunModeNone,
			wantErr:     false,
		},
		{
			name:        "deploy true, run dev",
			deploy:      true,
			run:         "dev",
			wantDeploy:  true,
			wantRunMode: prompt.RunModeDev,
			wantErr:     false,
		},
		{
			name:        "deploy true, run dev-remote",
			deploy:      true,
			run:         "dev-remote",
			wantDeploy:  true,
			wantRunMode: prompt.RunModeDevRemote,
			wantErr:     false,
		},
		{
			name:        "deploy false, run dev-remote (error)",
			deploy:      false,
			run:         "dev-remote",
			wantDeploy:  false,
			wantRunMode: prompt.RunModeNone,
			wantErr:     true,
		},
		{
			name:        "empty run value",
			deploy:      false,
			run:         "",
			wantDeploy:  false,
			wantRunMode: prompt.RunModeNone,
			wantErr:     false,
		},
		{
			name:        "invalid run value",
			deploy:      true,
			run:         "invalid",
			wantDeploy:  false,
			wantRunMode: prompt.RunModeNone,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deploy, runMode, err := parseDeployAndRunFlags(tt.deploy, tt.run)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantDeploy, deploy)
			assert.Equal(t, tt.wantRunMode, runMode)
		})
	}
}

func TestResourceBindingsBuilder(t *testing.T) {
	tests := []struct {
		name     string
		addOps   func(*resourceBindingsBuilder)
		expected string
	}{
		{
			name: "warehouse only",
			addOps: func(b *resourceBindingsBuilder) {
				b.addWarehouse("abc123")
			},
			expected: "        - name: sql-warehouse\n          description: SQL Warehouse for analytics\n          sql_warehouse:\n            id: ${var.warehouse_id}\n            permission: CAN_USE",
		},
		{
			name: "multiple resources",
			addOps: func(b *resourceBindingsBuilder) {
				b.addWarehouse("wh1")
				b.addServingEndpoint("ep1")
				b.addExperiment("exp1")
			},
			expected: "        - name: sql-warehouse\n          description: SQL Warehouse for analytics\n          sql_warehouse:\n            id: ${var.warehouse_id}\n            permission: CAN_USE\n        - name: serving-endpoint\n          description: Model serving endpoint\n          serving_endpoint:\n            name: ${var.serving_endpoint_name}\n            permission: CAN_QUERY\n        - name: experiment\n          description: MLflow experiment\n          experiment:\n            experiment_id: ${var.experiment_id}\n            permission: CAN_MANAGE",
		},
		{
			name: "empty resources",
			addOps: func(b *resourceBindingsBuilder) {
				b.addWarehouse("")
				b.addServingEndpoint("")
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := newResourceBindingsBuilder()
			tt.addOps(builder)
			result := builder.build()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasResourceSpec(t *testing.T) {
	tests := []struct {
		name     string
		manifest *appTemplateManifest
		checker  resourceSpecChecker
		expected bool
	}{
		{
			name: "has SQL warehouse",
			manifest: &appTemplateManifest{
				Manifest: manifest{
					ResourceSpecs: []resourceSpec{
						{Name: "warehouse", SQLWarehouseSpec: &map[string]any{"id": "abc"}},
					},
				},
			},
			checker:  func(s *resourceSpec) bool { return s.SQLWarehouseSpec != nil },
			expected: true,
		},
		{
			name: "no SQL warehouse",
			manifest: &appTemplateManifest{
				Manifest: manifest{
					ResourceSpecs: []resourceSpec{
						{Name: "endpoint", ServingEndpointSpec: &map[string]any{"name": "ep"}},
					},
				},
			},
			checker:  func(s *resourceSpec) bool { return s.SQLWarehouseSpec != nil },
			expected: false,
		},
		{
			name: "empty specs",
			manifest: &appTemplateManifest{
				Manifest: manifest{
					ResourceSpecs: []resourceSpec{},
				},
			},
			checker:  func(s *resourceSpec) bool { return s.SQLWarehouseSpec != nil },
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasResourceSpec(tt.manifest, tt.checker)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatabricksYmlTemplateWithUserAPIScopes(t *testing.T) {
	tests := []struct {
		name          string
		vars          templateVars
		expectScopes  bool
		expectedScope string
	}{
		{
			name: "with user_api_scopes",
			vars: templateVars{
				ProjectName:    "test-app",
				AppName:        "test-app",
				AppDescription: "Test application",
				UserAPIScopes:  []string{"sql", "catalog.connections"},
			},
			expectScopes:  true,
			expectedScope: "user_api_scopes:\n        - sql\n        - catalog.connections",
		},
		{
			name: "without user_api_scopes",
			vars: templateVars{
				ProjectName:    "test-app",
				AppName:        "test-app",
				AppDescription: "Test application",
				UserAPIScopes:  nil,
			},
			expectScopes: false,
		},
		{
			name: "with empty user_api_scopes",
			vars: templateVars{
				ProjectName:    "test-app",
				AppName:        "test-app",
				AppDescription: "Test application",
				UserAPIScopes:  []string{},
			},
			expectScopes: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpl, err := template.New("databricks.yml").Parse(databricksYmlTemplate)
			require.NoError(t, err)

			var buf bytes.Buffer
			err = tmpl.Execute(&buf, tt.vars)
			require.NoError(t, err)

			output := buf.String()

			if tt.expectScopes {
				assert.Contains(t, output, tt.expectedScope)
			} else {
				assert.NotContains(t, output, "user_api_scopes")
			}

			// Verify basic structure is present
			assert.Contains(t, output, "bundle:")
			assert.Contains(t, output, "name: test-app")
			assert.Contains(t, output, "resources:")
			assert.Contains(t, output, "apps:")
		})
	}
}

func TestUserAPIScopesInLegacyManifest(t *testing.T) {
	// Verify that the manifest JSON includes templates with user_api_scopes
	manifests, err := loadLegacyTemplates()
	require.NoError(t, err)
	require.NotNil(t, manifests)

	// Find a template with user_api_scopes (e.g., dash-data-app-obo-user)
	var foundWithScopes bool
	for i := range manifests.Templates {
		tmpl := &manifests.Templates[i]
		if strings.Contains(tmpl.Path, "obo-user") {
			foundWithScopes = true
			assert.NotEmpty(t, tmpl.Manifest.UserAPIScopes, "Template %s should have user_api_scopes", tmpl.Path)
			assert.Contains(t, tmpl.Manifest.UserAPIScopes, "sql", "Template %s should include 'sql' scope", tmpl.Path)
		}
	}

	assert.True(t, foundWithScopes, "Should have found at least one template with user_api_scopes")
}
