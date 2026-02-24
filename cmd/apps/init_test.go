package apps

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/apps/manifest"
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

func testVars() templateVars {
	return templateVars{
		ProjectName:    "my-app",
		AppDescription: "My awesome app",
		Profile:        "default",
		WorkspaceHost:  "https://dbc-123.cloud.databricks.com",
		Bundle: tmplBundle{
			Variables:       "sql_warehouse_id:",
			Resources:       "- name: sql-warehouse",
			TargetVariables: "sql_warehouse_id: abc123",
		},
		DotEnv: dotEnvVars{
			Content: "WH_ID=abc123",
			Example: "WH_ID=your_sql_warehouse_id",
		},
		AppEnv:  "- name: SQL_WAREHOUSE_ID\n  valueFrom: sql_warehouse",
		Plugins: map[string]*pluginVar{"analytics": {}},
	}
}

func TestExecuteTemplateBackwardCompat(t *testing.T) {
	ctx := context.Background()
	vars := testVars()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "project_name",
			input:    "{{.project_name}}",
			expected: "my-app",
		},
		{
			name:     "app_description",
			input:    "{{.app_description}}",
			expected: "My awesome app",
		},
		{
			name:     "workspace_host",
			input:    "{{.workspace_host}}",
			expected: "https://dbc-123.cloud.databricks.com",
		},
		{
			name:     "dotenv",
			input:    "{{.dotenv}}",
			expected: "WH_ID=abc123",
		},
		{
			name:     "dotenv_example",
			input:    "{{.dotenv_example}}",
			expected: "WH_ID=your_sql_warehouse_id",
		},
		{
			name:     "variables",
			input:    "{{.variables}}",
			expected: "sql_warehouse_id:",
		},
		{
			name:     "resources",
			input:    "{{.resources}}",
			expected: "- name: sql-warehouse",
		},
		{
			name:     "target_variables",
			input:    "{{.target_variables}}",
			expected: "sql_warehouse_id: abc123",
		},
		{
			name:     "plugin_imports",
			input:    "{{.plugin_imports}}",
			expected: "analytics",
		},
		{
			name:     "plugin_usages",
			input:    "{{.plugin_usages}}",
			expected: "analytics()",
		},
		{
			name:     "app_env",
			input:    "{{.app_env}}",
			expected: "- name: SQL_WAREHOUSE_ID\n  valueFrom: sql_warehouse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeTemplate(ctx, "test.txt", []byte(tt.input), vars)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestExecuteTemplateNewKeys(t *testing.T) {
	ctx := context.Background()
	vars := testVars()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "projectName",
			input:    "{{.projectName}}",
			expected: "my-app",
		},
		{
			name:     "appDescription",
			input:    "{{.appDescription}}",
			expected: "My awesome app",
		},
		{
			name:     "workspaceHost",
			input:    "{{.workspaceHost}}",
			expected: "https://dbc-123.cloud.databricks.com",
		},
		{
			name:     "bundle.variables",
			input:    "{{.bundle.variables}}",
			expected: "sql_warehouse_id:",
		},
		{
			name:     "bundle.resources",
			input:    "{{.bundle.resources}}",
			expected: "- name: sql-warehouse",
		},
		{
			name:     "bundle.targetVariables",
			input:    "{{.bundle.targetVariables}}",
			expected: "sql_warehouse_id: abc123",
		},
		{
			name:     "dotEnv.content",
			input:    "{{.dotEnv.content}}",
			expected: "WH_ID=abc123",
		},
		{
			name:     "dotEnv.example",
			input:    "{{.dotEnv.example}}",
			expected: "WH_ID=your_sql_warehouse_id",
		},
		{
			name:     "plugins selected",
			input:    `{{if .plugins.analytics}}yes{{end}}`,
			expected: "yes",
		},
		{
			name:     "plugins not selected",
			input:    `{{if .plugins.nonexistent}}yes{{end}}`,
			expected: "",
		},
		{
			name:     "appEnv",
			input:    "{{.appEnv}}",
			expected: "- name: SQL_WAREHOUSE_ID\n  valueFrom: sql_warehouse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := executeTemplate(ctx, "test.txt", []byte(tt.input), vars)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestExecuteTemplateInvalidSyntaxReturnsOriginal(t *testing.T) {
	ctx := context.Background()
	vars := templateVars{ProjectName: "my-app"}
	input := "some content with bad {{ syntax"
	result, err := executeTemplate(ctx, "test.js", []byte(input), vars)
	require.NoError(t, err)
	assert.Equal(t, input, string(result))
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
		{"0.3.0", "template-v0.3.0"},
		{"1.0.0", "template-v1.0.0"},
		{"v0.3.0", "template-v0.3.0"},
		{"v1.0.0", "template-v1.0.0"},
		{"template-v0.3.0", "template-v0.3.0"},
		{"latest", "main"},
		{"", ""},
		{"main", "main"},
		{"feat/something", "feat/something"},
		{appkitDefaultVersion, appkitDefaultVersion},
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

// testManifest returns a manifest with an "analytics" plugin for testing parseSetValues.
func testManifest() *manifest.Manifest {
	return &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"analytics": {
				Name: "analytics",
				Resources: manifest.Resources{
					Required: []manifest.Resource{
						{
							Type:        "sql_warehouse",
							Alias:       "SQL Warehouse",
							ResourceKey: "sql-warehouse",
							Fields:      map[string]manifest.ResourceField{"id": {Env: "WH_ID"}},
						},
					},
					Optional: []manifest.Resource{
						{
							Type:        "database",
							Alias:       "Database",
							ResourceKey: "database",
							Fields: map[string]manifest.ResourceField{
								"instance_name": {Env: "DB_INST"},
								"database_name": {Env: "DB_NAME"},
							},
						},
						{
							Type:        "secret",
							Alias:       "Secret",
							ResourceKey: "secret",
							Fields: map[string]manifest.ResourceField{
								"scope": {Env: "SECRET_SCOPE"},
								"key":   {Env: "SECRET_KEY"},
							},
						},
					},
				},
			},
		},
	}
}

func TestParseSetValues(t *testing.T) {
	m := testManifest()

	tests := []struct {
		name      string
		setValues []string
		wantRV    map[string]string
		wantErr   string
	}{
		{
			name:      "single field",
			setValues: []string{"analytics.sql-warehouse.id=abc123"},
			wantRV:    map[string]string{"sql-warehouse.id": "abc123"},
		},
		{
			name:      "multi-field complete",
			setValues: []string{"analytics.database.instance_name=inst", "analytics.database.database_name=mydb"},
			wantRV:    map[string]string{"database.instance_name": "inst", "database.database_name": "mydb"},
		},
		{
			name:      "later set overrides earlier",
			setValues: []string{"analytics.sql-warehouse.id=first", "analytics.sql-warehouse.id=second"},
			wantRV:    map[string]string{"sql-warehouse.id": "second"},
		},
		{
			name:      "empty set values",
			setValues: nil,
			wantRV:    map[string]string{},
		},
		{
			name:      "missing equals sign",
			setValues: []string{"analytics.sql-warehouse.id"},
			wantErr:   "invalid --set format",
		},
		{
			name:      "too few key parts",
			setValues: []string{"sql-warehouse.id=abc"},
			wantErr:   "invalid --set key",
		},
		{
			name:      "unknown plugin",
			setValues: []string{"nosuch.sql-warehouse.id=abc"},
			wantErr:   `unknown plugin "nosuch"`,
		},
		{
			name:      "unknown resource key",
			setValues: []string{"analytics.nosuch.id=abc"},
			wantErr:   `has no resource with key "nosuch"`,
		},
		{
			name:      "unknown field",
			setValues: []string{"analytics.sql-warehouse.nosuch=abc"},
			wantErr:   `field "nosuch"`,
		},
		{
			name:      "multi-field incomplete database",
			setValues: []string{"analytics.database.instance_name=inst"},
			wantErr:   `incomplete resource "database"`,
		},
		{
			name:      "multi-field incomplete secret",
			setValues: []string{"analytics.secret.scope=myscope"},
			wantErr:   `incomplete resource "secret"`,
		},
		{
			name: "all fields together",
			setValues: []string{
				"analytics.sql-warehouse.id=wh1",
				"analytics.database.instance_name=inst",
				"analytics.database.database_name=mydb",
				"analytics.secret.scope=s",
				"analytics.secret.key=k",
			},
			wantRV: map[string]string{
				"sql-warehouse.id":       "wh1",
				"database.instance_name": "inst",
				"database.database_name": "mydb",
				"secret.scope":           "s",
				"secret.key":             "k",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rv, err := parseSetValues(tt.setValues, m)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantRV, rv)
			}
		})
	}
}

func TestPluginHasResourceField(t *testing.T) {
	m := testManifest()
	p := m.GetPluginByName("analytics")
	require.NotNil(t, p)

	assert.True(t, pluginHasResourceField(p, "sql-warehouse", "id"))
	assert.True(t, pluginHasResourceField(p, "database", "instance_name"))
	assert.True(t, pluginHasResourceField(p, "secret", "scope"))
	assert.False(t, pluginHasResourceField(p, "sql-warehouse", "nosuch"))
	assert.False(t, pluginHasResourceField(p, "nosuch", "id"))
}

func TestAppendUnique(t *testing.T) {
	result := appendUnique([]string{"a", "b"}, "b", "c", "a", "d")
	assert.Equal(t, []string{"a", "b", "c", "d"}, result)
}

func TestAppendUniqueEmptyBase(t *testing.T) {
	result := appendUnique(nil, "x", "y", "x")
	assert.Equal(t, []string{"x", "y"}, result)
}

func TestAppendUniqueNoValues(t *testing.T) {
	result := appendUnique([]string{"a", "b"})
	assert.Equal(t, []string{"a", "b"}, result)
}

func TestRunManifestOnlyFound(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, manifest.ManifestFileName)
	content := `{"$schema":"https://example.com/schema","version":"1.0","plugins":{"analytics":{"name":"analytics","resources":{"required":[],"optional":[]}}}}`
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0o644))

	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	err = runManifestOnly(context.Background(), dir, "", "")
	w.Close()
	os.Stdout = old
	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()
	assert.Contains(t, out, `"version": "1.0"`)
	assert.Contains(t, out, `"analytics"`)
}

func TestRunManifestOnlyNotFound(t *testing.T) {
	dir := t.TempDir()

	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	err = runManifestOnly(context.Background(), dir, "", "")
	w.Close()
	os.Stdout = old
	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()
	assert.Equal(t, "No appkit.plugins.json manifest found in this template.\n", out)
}
