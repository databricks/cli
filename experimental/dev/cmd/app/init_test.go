package app

import (
	"testing"

	"github.com/databricks/cli/experimental/dev/lib/prompt"
	"github.com/stretchr/testify/assert"
)

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		wantRepoURL string
		wantSubdir  string
		wantBranch  string
	}{
		{
			name:        "simple repo URL",
			url:         "https://github.com/user/repo",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "",
			wantBranch:  "",
		},
		{
			name:        "repo URL with trailing slash",
			url:         "https://github.com/user/repo/",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "",
			wantBranch:  "",
		},
		{
			name:        "repo with branch",
			url:         "https://github.com/user/repo/tree/main",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "",
			wantBranch:  "main",
		},
		{
			name:        "repo with branch and subdir",
			url:         "https://github.com/user/repo/tree/main/templates/starter",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "templates/starter",
			wantBranch:  "main",
		},
		{
			name:        "repo with branch and deep subdir",
			url:         "https://github.com/databricks/cli/tree/v0.1.0/libs/template/templates/default-python",
			wantRepoURL: "https://github.com/databricks/cli",
			wantSubdir:  "libs/template/templates/default-python",
			wantBranch:  "v0.1.0",
		},
		{
			name:        "repo with feature branch",
			url:         "https://github.com/user/repo/tree/feature/my-feature",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "my-feature",
			wantBranch:  "feature",
		},
		{
			name:        "repo URL with trailing slash and tree",
			url:         "https://github.com/user/repo/tree/main/",
			wantRepoURL: "https://github.com/user/repo",
			wantSubdir:  "",
			wantBranch:  "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRepoURL, gotSubdir, gotBranch := parseGitHubURL(tt.url)
			assert.Equal(t, tt.wantRepoURL, gotRepoURL, "repoURL mismatch")
			assert.Equal(t, tt.wantSubdir, gotSubdir, "subdir mismatch")
			assert.Equal(t, tt.wantBranch, gotBranch, "branch mismatch")
		})
	}
}

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
			name:        "deploy false, run dev-remote",
			deploy:      false,
			run:         "dev-remote",
			wantDeploy:  false,
			wantRunMode: prompt.RunModeDevRemote,
			wantErr:     false,
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
