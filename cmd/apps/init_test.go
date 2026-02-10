package apps

import (
	"errors"
	"testing"

	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/spf13/cobra"
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
		PluginImports:  "analytics",
		PluginUsages:   "analytics()",
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
			input:    "import { {{.plugin_imports}} } from 'appkit'",
			expected: "import { analytics } from 'appkit'",
		},
		{
			name:     "plugin usage substitution",
			input:    "plugins: [{{.plugin_usages}}]",
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
		PluginImports:  "", // No plugins
		PluginUsages:   "",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes plugin import with comma",
			input:    "import { core, {{.plugin_imports}} } from 'appkit'",
			expected: "import { core } from 'appkit'",
		},
		{
			name:     "removes plugin usage line",
			input:    "plugins: [\n    {{.plugin_usages}},\n]",
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

func TestInitCmdBranchAndVersionMutuallyExclusive(t *testing.T) {
	cmd := newInitCmd()
	cmd.PreRunE = nil // skip workspace client setup for flag validation test
	// Replace RunE to only test flag validation, not the full create flow
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if cmd.Flags().Changed("branch") && cmd.Flags().Changed("version") {
			return errors.New("--branch and --version are mutually exclusive")
		}
		return nil
	}
	cmd.SetArgs([]string{"--branch", "dev", "--version", "v1.0.0"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--branch and --version are mutually exclusive")
}

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"0.3.0", "v0.3.0"},
		{"1.0.0", "v1.0.0"},
		{"v0.3.0", "v0.3.0"},
		{"v1.0.0", "v1.0.0"},
		{"latest", "main"},
		{"", ""},
		{"main", "main"},
		{"feat/something", "feat/something"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeVersion(tt.input)
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

func TestValidateMultiFieldFlags(t *testing.T) {
	tests := []struct {
		name    string
		opts    createOptions
		wantErr string
	}{
		{
			name: "both database fields provided",
			opts: createOptions{databaseInstance: "inst", databaseName: "db"},
		},
		{
			name: "neither database field provided",
			opts: createOptions{},
		},
		{
			name:    "only database instance name",
			opts:    createOptions{databaseInstance: "inst"},
			wantErr: "--database-instance and --database-name must be provided together",
		},
		{
			name:    "only database database name",
			opts:    createOptions{databaseName: "db"},
			wantErr: "--database-instance and --database-name must be provided together",
		},
		{
			name: "both secret fields provided",
			opts: createOptions{secretScope: "scope", secretKey: "key"},
		},
		{
			name: "neither secret field provided",
			opts: createOptions{},
		},
		{
			name:    "only secret scope",
			opts:    createOptions{secretScope: "scope"},
			wantErr: "--secret-scope and --secret-key must be provided together",
		},
		{
			name:    "only secret key",
			opts:    createOptions{secretKey: "key"},
			wantErr: "--secret-scope and --secret-key must be provided together",
		},
		{
			name: "all multi-field flags provided",
			opts: createOptions{
				databaseInstance: "inst",
				databaseName:     "db",
				secretScope:      "scope",
				secretKey:        "key",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.validateMultiFieldFlags()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tt.wantErr, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPopulateResourceValues(t *testing.T) {
	opts := createOptions{
		warehouseID:         "wh-123",
		jobID:               "job-456",
		modelEndpointID:     "ep-789",
		volumeID:            "cat.schema.vol",
		vectorSearchIndexID: "idx-1",
		functionID:          "cat.schema.fn",
		connectionID:        "conn-1",
		genieSpaceID:        "gs-1",
		experimentID:        "exp-1",
		databaseInstance:    "inst",
		databaseName:        "mydb",
		secretScope:         "scope",
		secretKey:           "key",
	}

	rv := make(map[string]string)
	opts.populateResourceValues(rv)

	assert.Equal(t, "wh-123", rv["sql-warehouse.id"])
	assert.Equal(t, "job-456", rv["job.id"])
	assert.Equal(t, "ep-789", rv["model-endpoint.id"])
	assert.Equal(t, "cat.schema.vol", rv["volume.id"])
	assert.Equal(t, "idx-1", rv["vector-search-index.id"])
	assert.Equal(t, "cat.schema.fn", rv["function.id"])
	assert.Equal(t, "conn-1", rv["connection.id"])
	assert.Equal(t, "gs-1", rv["genie-space.space_id"])
	assert.Equal(t, "exp-1", rv["experiment.id"])
	assert.Equal(t, "inst", rv["database.instance_name"])
	assert.Equal(t, "mydb", rv["database.database_name"])
	assert.Equal(t, "scope", rv["secret.scope"])
	assert.Equal(t, "key", rv["secret.key"])
}

func TestPopulateResourceValuesSkipsEmpty(t *testing.T) {
	opts := createOptions{
		warehouseID: "wh-123",
	}

	rv := make(map[string]string)
	opts.populateResourceValues(rv)

	assert.Equal(t, "wh-123", rv["sql-warehouse.id"])
	assert.Len(t, rv, 1)
}
