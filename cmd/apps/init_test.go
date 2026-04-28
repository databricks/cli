package apps

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/apps/manifest"
	"github.com/databricks/cli/libs/apps/prompt"
	"github.com/databricks/cli/libs/env"
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
	ctx := t.Context()
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
	ctx := t.Context()
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
	ctx := t.Context()
	vars := templateVars{ProjectName: "my-app"}
	input := "some content with bad {{ syntax"
	result, err := executeTemplate(ctx, "test.js", []byte(input), vars)
	require.NoError(t, err)
	assert.Equal(t, input, string(result))
}

// TestExecuteTemplatePluginStability locks down the contract that the
// AppKit init template relies on: ranging over .plugins exposes a
// .Stability field per plugin, with stable/unset rendering as the empty
// string. See databricks/appkit#264 commit d826a532 (server.ts branches
// imports between `@databricks/appkit` and `@databricks/appkit/beta`).
func TestExecuteTemplatePluginStability(t *testing.T) {
	ctx := t.Context()
	vars := templateVars{
		Plugins: map[string]*pluginVar{
			"stable-plugin": {},
			"beta-plugin":   {Stability: "beta"},
		},
	}

	input := `{{range $n, $p := .plugins}}{{$n}}={{$p.Stability}};{{end}}`
	result, err := executeTemplate(ctx, "server.ts", []byte(input), vars)
	require.NoError(t, err)
	got := string(result)

	assert.Contains(t, got, "stable-plugin=;")
	assert.Contains(t, got, "beta-plugin=beta;")
}

// TestExecuteTemplateBetaImportAccumulator pins the full text/template
// pattern used by the AppKit server.ts template (databricks/appkit#264
// commit 488797fc): a string-accumulator pre-pass over .plugins that
// reassigns an outer-scope variable inside `range` and concatenates
// names via `printf`, then emits a single guarded import line.
//
// If a future refactor of executeTemplate breaks variable reassignment,
// printf, or pointer-field access on map values, this test fails before
// users see broken init output.
func TestExecuteTemplateBetaImportAccumulator(t *testing.T) {
	ctx := t.Context()

	// Mirror of the relevant slice of template/server/server.ts in AppKit.
	// Kept as a literal string (not loaded from the AppKit repo) so this
	// test is hermetic and survives AppKit branch movement.
	input := `{{- $betaImports := "" -}}
{{- range $name, $p := .plugins -}}
  {{- if eq $p.Stability "beta" -}}
    {{- if eq $betaImports "" -}}
      {{- $betaImports = $name -}}
    {{- else -}}
      {{- $betaImports = printf "%s, %s" $betaImports $name -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
import { createApp{{range $name, $p := .plugins}}{{if ne $p.Stability "beta"}}, {{$name}}{{end}}{{end}} } from '@databricks/appkit';
{{- if ne $betaImports "" }}
import { {{$betaImports}} } from '@databricks/appkit/beta';
{{- end}}
`

	cases := []struct {
		name              string
		plugins           map[string]*pluginVar
		wantStableImports []string // names that must appear on the stable line
		wantBetaImports   []string // names that must appear on the beta line, "" means no beta line
		wantNoBetaLine    bool
	}{
		{
			name: "all stable: no beta line",
			plugins: map[string]*pluginVar{
				"server":    {},
				"analytics": {},
			},
			wantStableImports: []string{"server", "analytics"},
			wantNoBetaLine:    true,
		},
		{
			name: "mixed single beta",
			plugins: map[string]*pluginVar{
				"server":  {},
				"betaOne": {Stability: "beta"},
			},
			wantStableImports: []string{"server"},
			wantBetaImports:   []string{"betaOne"},
		},
		{
			name: "mixed multiple betas: combined into one import line",
			plugins: map[string]*pluginVar{
				"server":  {},
				"betaOne": {Stability: "beta"},
				"betaTwo": {Stability: "beta"},
			},
			wantStableImports: []string{"server"},
			wantBetaImports:   []string{"betaOne", "betaTwo"},
		},
		{
			name: "all beta: createApp alone on stable line",
			plugins: map[string]*pluginVar{
				"betaOne": {Stability: "beta"},
				"betaTwo": {Stability: "beta"},
			},
			wantBetaImports: []string{"betaOne", "betaTwo"},
		},
		{
			name: "future tier (alpha) routes to stable line for now",
			plugins: map[string]*pluginVar{
				"server":   {},
				"alphaOne": {Stability: "alpha"},
			},
			wantStableImports: []string{"server", "alphaOne"},
			wantNoBetaLine:    true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vars := templateVars{Plugins: tc.plugins}
			result, err := executeTemplate(ctx, "server.ts", []byte(input), vars)
			require.NoError(t, err)
			got := string(result)

			lines := strings.Split(got, "\n")
			require.NotEmpty(t, lines)

			// Stable line is always first: starts with `import { createApp`
			// and ends with `from '@databricks/appkit';`.
			stableLine := lines[0]
			assert.True(t, strings.HasPrefix(stableLine, "import { createApp"),
				"stable line: %q", stableLine)
			assert.True(t, strings.HasSuffix(stableLine, "} from '@databricks/appkit';"),
				"stable line: %q", stableLine)
			for _, name := range tc.wantStableImports {
				assert.Contains(t, stableLine, name,
					"stable line missing %q: %q", name, stableLine)
			}
			for _, name := range tc.wantBetaImports {
				assert.NotContains(t, stableLine, ", "+name,
					"beta plugin %q leaked onto stable line: %q", name, stableLine)
			}

			if tc.wantNoBetaLine {
				assert.NotContains(t, got, "@databricks/appkit/beta",
					"unexpected beta import emitted: %q", got)
				return
			}

			// Beta line: exactly one `from '@databricks/appkit/beta'` line.
			betaLineCount := strings.Count(got, "from '@databricks/appkit/beta'")
			assert.Equal(t, 1, betaLineCount,
				"expected exactly one beta import line, got %d: %q", betaLineCount, got)

			var betaLine string
			for _, l := range lines {
				if strings.Contains(l, "@databricks/appkit/beta") {
					betaLine = l
					break
				}
			}
			require.NotEmpty(t, betaLine, "beta line not found in: %q", got)
			for _, name := range tc.wantBetaImports {
				assert.Contains(t, betaLine, name,
					"beta line missing %q: %q", name, betaLine)
			}
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

func TestParseSetValuesBundleIgnoreSkipped(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"lakebase": {
				Name: "lakebase",
				Resources: manifest.Resources{
					Required: []manifest.Resource{
						{
							Type:        "postgres",
							Alias:       "Postgres",
							ResourceKey: "postgres",
							Fields: map[string]manifest.ResourceField{
								"branch":       {Description: "branch path"},
								"database":     {Description: "database name"},
								"endpointPath": {Env: "LAKEBASE_ENDPOINT", BundleIgnore: true},
							},
						},
					},
				},
			},
		},
	}

	rv, err := parseSetValues([]string{
		"lakebase.postgres.branch=projects/p1/branches/main",
		"lakebase.postgres.database=mydb",
	}, m)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"postgres.branch":   "projects/p1/branches/main",
		"postgres.database": "mydb",
	}, rv)

	// Setting only one non-bundleIgnore field should still fail.
	_, err = parseSetValues([]string{"lakebase.postgres.branch=br"}, m)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `incomplete resource "postgres"`)

	// bundleIgnore field can still be set explicitly via --set.
	rv, err = parseSetValues([]string{
		"lakebase.postgres.branch=br",
		"lakebase.postgres.database=db",
		"lakebase.postgres.endpointPath=ep",
	}, m)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"postgres.branch":       "br",
		"postgres.database":     "db",
		"postgres.endpointPath": "ep",
	}, rv)
}

func TestParseSetValuesLocalOnlySkipped(t *testing.T) {
	m := &manifest.Manifest{
		Plugins: map[string]manifest.Plugin{
			"lakebase": {
				Name: "lakebase",
				Resources: manifest.Resources{
					Required: []manifest.Resource{
						{
							Type:        "postgres",
							Alias:       "Postgres",
							ResourceKey: "postgres",
							Fields: map[string]manifest.ResourceField{
								"branch":       {Description: "branch path"},
								"database":     {Description: "database name"},
								"host":         {Env: "PGHOST", LocalOnly: true, Resolve: "postgres:host"},
								"databaseName": {Env: "PGDATABASE", LocalOnly: true, Resolve: "postgres:databaseName"},
								"endpointPath": {Env: "LAKEBASE_ENDPOINT", BundleIgnore: true, Resolve: "postgres:endpointPath"},
								"port":         {Env: "PGPORT", LocalOnly: true, Value: "5432"},
								"sslmode":      {Env: "PGSSLMODE", LocalOnly: true, Value: "require"},
							},
						},
					},
				},
			},
		},
	}

	// Setting only branch+database should succeed — localOnly and bundleIgnore fields are exempt.
	rv, err := parseSetValues([]string{
		"lakebase.postgres.branch=projects/p1/branches/main",
		"lakebase.postgres.database=mydb",
	}, m)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{
		"postgres.branch":   "projects/p1/branches/main",
		"postgres.database": "mydb",
	}, rv)

	// Setting only branch should still fail (database is also required).
	_, err = parseSetValues([]string{"lakebase.postgres.branch=br"}, m)
	require.Error(t, err)
	assert.Contains(t, err.Error(), `incomplete resource "postgres"`)
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

	err = runManifestOnly(t.Context(), dir, "", "")
	w.Close()
	os.Stdout = old
	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()
	assert.Equal(t, content, out)
}

func TestRunManifestOnlyNotFound(t *testing.T) {
	dir := t.TempDir()

	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	err = runManifestOnly(t.Context(), dir, "", "")
	w.Close()
	os.Stdout = old
	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()
	assert.Equal(t, "No appkit.plugins.json manifest found in this template.\n", out)
}

func TestRunManifestOnlyUsesTemplatePathEnvVar(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, manifest.ManifestFileName)
	content := `{"version":"1.0","scaffolding":{"command":"databricks apps init"}}`
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0o644))

	old := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w

	ctx := env.Set(t.Context(), templatePathEnvVar, dir)
	err = runManifestOnly(ctx, "", "", "")
	w.Close()
	os.Stdout = old
	require.NoError(t, err)

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()
	assert.Equal(t, content, out)
}

func TestCopyFileDeps(t *testing.T) {
	ctx := t.Context()

	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create a fake tarball in srcDir
	tgzContent := []byte("fake-tarball-content")
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "my-pkg-1.0.0.tgz"), tgzContent, 0o644))

	// package.json with file: dep, a registry dep, and a devDep with file:
	pkgJSON := []byte(`{
		"dependencies": {
			"my-pkg": "file:./my-pkg-1.0.0.tgz",
			"lodash": "4.17.21"
		},
		"devDependencies": {
			"missing-pkg": "file:./nonexistent.tgz"
		}
	}`)

	copyFileDeps(ctx, pkgJSON, srcDir, destDir)

	// The file: dep should be copied
	copied, err := os.ReadFile(filepath.Join(destDir, "my-pkg-1.0.0.tgz"))
	require.NoError(t, err)
	assert.Equal(t, tgzContent, copied)

	// The registry dep should NOT create any file
	_, err = os.Stat(filepath.Join(destDir, "4.17.21"))
	assert.ErrorIs(t, err, fs.ErrNotExist)

	// The missing file: dep should be skipped gracefully (no panic, no error)
	_, err = os.Stat(filepath.Join(destDir, "nonexistent.tgz"))
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestCopyFileDepsInvalidJSON(t *testing.T) {
	ctx := t.Context()
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Should not panic on invalid JSON
	copyFileDeps(ctx, []byte("not json"), srcDir, destDir)

	// destDir should remain empty
	entries, err := os.ReadDir(destDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestCopyFileDepsNoDeps(t *testing.T) {
	ctx := t.Context()
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// package.json with no file: deps
	pkgJSON := []byte(`{"dependencies": {"react": "19.0.0"}}`)
	copyFileDeps(ctx, pkgJSON, srcDir, destDir)

	entries, err := os.ReadDir(destDir)
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func skipIfNoNpm(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not found in PATH, skipping")
	}
}

func TestStartBackgroundNpmInstall_NoLockFile(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Only package.json, no lock file
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "package.json"), []byte(`{"name":"test"}`), 0o644))

	ch := startBackgroundNpmInstall(t.Context(), srcDir, destDir, "test-app")
	assert.Nil(t, ch)
}

func TestStartBackgroundNpmInstall_NoPackageJSON(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Only lock file, no package.json
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "package-lock.json"), []byte(`{}`), 0o644))

	ch := startBackgroundNpmInstall(t.Context(), srcDir, destDir, "test-app")
	assert.Nil(t, ch)
}

func TestStartBackgroundNpmInstall_CopiesFiles(t *testing.T) {
	skipIfNoNpm(t)

	srcDir := t.TempDir()
	destDir := filepath.Join(t.TempDir(), "output")

	pkgJSON := []byte(`{"name":"{{.projectName}}","version":"1.0.0"}`)
	lockJSON := []byte(`{"lockfileVersion":3,"packages":{}}`)
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "package.json"), pkgJSON, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "package-lock.json"), lockJSON, 0o644))

	ch := startBackgroundNpmInstall(t.Context(), srcDir, destDir, "my-app")
	require.NotNil(t, ch)

	// Drain the channel to avoid goroutine leak (npm ci will fail on fake data)
	<-ch

	// package.json should be written with template substitution
	got, err := os.ReadFile(filepath.Join(destDir, "package.json"))
	require.NoError(t, err)
	assert.Contains(t, string(got), `"my-app"`)
	assert.NotContains(t, string(got), "{{.projectName}}")

	// package-lock.json should be copied verbatim
	gotLock, err := os.ReadFile(filepath.Join(destDir, "package-lock.json"))
	require.NoError(t, err)
	assert.Equal(t, lockJSON, gotLock)
}

func TestStartBackgroundNpmInstall_CopiesFileDeps(t *testing.T) {
	skipIfNoNpm(t)

	srcDir := t.TempDir()
	destDir := filepath.Join(t.TempDir(), "output")

	tgzContent := []byte("fake-tarball")
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "my-pkg-1.0.0.tgz"), tgzContent, 0o644))

	pkgJSON := []byte(`{"name":"test","dependencies":{"my-pkg":"file:./my-pkg-1.0.0.tgz"}}`)
	lockJSON := []byte(`{"lockfileVersion":3,"packages":{}}`)
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "package.json"), pkgJSON, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "package-lock.json"), lockJSON, 0o644))

	ch := startBackgroundNpmInstall(t.Context(), srcDir, destDir, "test-app")
	require.NotNil(t, ch)
	<-ch

	// The file: dep tarball should be copied to destDir
	copied, err := os.ReadFile(filepath.Join(destDir, "my-pkg-1.0.0.tgz"))
	require.NoError(t, err)
	assert.Equal(t, tgzContent, copied)
}

func TestStartBackgroundNpmInstall_TemplateSubstitution(t *testing.T) {
	skipIfNoNpm(t)

	srcDir := t.TempDir()
	destDir := filepath.Join(t.TempDir(), "output")

	pkgJSON := []byte(`{"name":"{{.projectName}}","description":"{{.appDescription}}"}`)
	lockJSON := []byte(`{"lockfileVersion":3}`)
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "package.json"), pkgJSON, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "package-lock.json"), lockJSON, 0o644))

	ch := startBackgroundNpmInstall(t.Context(), srcDir, destDir, "cool-project")
	require.NotNil(t, ch)
	<-ch

	got, err := os.ReadFile(filepath.Join(destDir, "package.json"))
	require.NoError(t, err)
	assert.Contains(t, string(got), `"cool-project"`)
	assert.NotContains(t, string(got), "{{.projectName}}")
}
