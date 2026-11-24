package agents

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// agentWithConfigDir returns an Agent whose ConfigDir resolves to dir, using
// Claude Code's manifest reader (the format these tests write).
func agentWithConfigDir(dir string) *Agent {
	return &Agent{
		Name:           "test-agent",
		DisplayName:    "Test Agent",
		ConfigDir:      func(_ context.Context) (string, error) { return dir, nil },
		pluginInstalls: claudePluginInstalls,
	}
}

// writeManifest writes body to <configDir>/plugins/installed_plugins.json.
func writeManifest(t *testing.T, configDir, body string) {
	dir := filepath.Join(configDir, "plugins")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, pluginManifestFile), []byte(body), 0o600))
}

func TestDatabricksPluginVersion(t *testing.T) {
	tests := []struct {
		name        string
		body        string // manifest body; "" means no manifest file at all
		wantOK      bool
		wantVersion string
	}{
		{"no manifest", "", false, ""},
		{"databricks plugin present", `{"plugins":{"databricks@claude-plugins-official":[{"scope":"user","version":"0.2.6"}]}}`, true, "0.2.6"},
		{"other plugin only", `{"plugins":{"gopls-lsp@claude-plugins-official":[{"version":"1.0.0"}]}}`, false, ""},
		{"databricks among others", `{"plugins":{"gopls-lsp@m":[{"version":"1.0.0"}],"databricks@claude-plugins-official":[{"version":"0.3.0"}]}}`, true, "0.3.0"},
		{"highest version across scopes", `{"plugins":{"databricks@m":[{"scope":"user","version":"0.2.9"},{"scope":"project","version":"0.2.10"},{"scope":"local","version":"0.2.8"}]}}`, true, "0.2.10"},
		{"databricks-prefixed but distinct id does not match", `{"plugins":{"databricks-foo@m":[{"version":"9.9.9"}]}}`, false, ""},
		{"installed but no records", `{"plugins":{"databricks@m":[]}}`, true, ""},
		{"installed but unversioned record", `{"plugins":{"databricks@m":[{"scope":"user"}]}}`, true, ""},
		{"malformed json", `{not json`, false, ""},
		{"empty plugins map", `{"plugins":{}}`, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configDir := t.TempDir()
			if tt.body != "" {
				writeManifest(t, configDir, tt.body)
			}
			version, ok := agentWithConfigDir(configDir).DatabricksPluginVersion(t.Context())
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.wantVersion, version)
		})
	}
}

func TestDatabricksPluginVersionAcrossMarketplaces(t *testing.T) {
	configDir := t.TempDir()
	writeManifest(t, configDir, `{"plugins":{"databricks@a":[{"scope":"user","version":"0.2.9"}],"databricks@b":[{"scope":"user","version":"0.3.0"}]}}`)
	a := agentWithConfigDir(configDir)
	installs, ok, err := claudePluginInstalls(t.Context(), a)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Len(t, installs, 2)
	version, ok := a.DatabricksPluginVersion(t.Context())
	require.True(t, ok)
	assert.Equal(t, "0.3.0", version)
}

// projectFixture creates <root>/Project/sub and returns the project dir and the
// sub dir, with root resolved so the paths are the ones the OS reports.
func projectFixture(t *testing.T) (project, sub string) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	require.NoError(t, err)
	project = filepath.Join(root, "Project")
	sub = filepath.Join(project, "sub")
	require.NoError(t, os.MkdirAll(sub, 0o755))
	return project, sub
}

// Every record counts, not only the first one the reader returns.
func TestDatabricksPluginActiveInChecksEveryRecord(t *testing.T) {
	project, _ := projectFixture(t)
	a := &Agent{pluginInstalls: func(context.Context, *Agent) ([]pluginInstall, bool, error) {
		return []pluginInstall{
			{Scope: "project", ProjectPath: filepath.Join(filepath.Dir(project), "Other")},
			{Scope: "user"},
		}, true, nil
	}}
	active, err := a.DatabricksPluginActiveIn(t.Context(), project)
	require.NoError(t, err)
	assert.True(t, active)
}

// A working directory spelled through a symlink inside the project that points
// out of it is the directory it points to, which the project's plugin does not
// cover.
func TestDatabricksPluginActiveInResolvesTheWorkingDir(t *testing.T) {
	project, _ := projectFixture(t)
	outside := filepath.Join(filepath.Dir(project), "Other")
	require.NoError(t, os.Mkdir(outside, 0o755))
	link := filepath.Join(project, "link")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("cannot create a symlink: %v", err)
	}
	a := &Agent{pluginInstalls: func(context.Context, *Agent) ([]pluginInstall, bool, error) {
		return []pluginInstall{{Scope: "project", ProjectPath: project}}, true, nil
	}}
	active, err := a.DatabricksPluginActiveIn(t.Context(), link)
	require.NoError(t, err)
	assert.False(t, active)
	active, err = a.DatabricksPluginActiveIn(t.Context(), project)
	require.NoError(t, err)
	assert.True(t, active)
}

// A missing manifest means no plugin; one that exists but cannot be parsed
// leaves the answer unknown, so the caller can consult another source.
func TestDatabricksPluginActiveInTellsAMissingManifestFromAnUnreadableOne(t *testing.T) {
	configDir := t.TempDir()
	a := agentWithConfigDir(configDir)
	active, err := a.DatabricksPluginActiveIn(t.Context(), "")
	require.NoError(t, err)
	assert.False(t, active)

	writeManifest(t, configDir, `{not json`)
	_, err = a.DatabricksPluginActiveIn(t.Context(), "")
	assert.Error(t, err)
}

func TestPluginInstallAppliesToProjectOutsideWorkingDir(t *testing.T) {
	project, sub := projectFixture(t)
	sibling := filepath.Join(filepath.Dir(project), "Other")
	require.NoError(t, os.Mkdir(sibling, 0o755))

	// Existing directories, so the comparison by file identity runs too.
	for _, cwd := range []string{sibling, filepath.Dir(project)} {
		assert.False(t, pluginInstall{Scope: "project", ProjectPath: project}.appliesTo(cwd), cwd)
	}
	assert.False(t, pluginInstall{Scope: "local", ProjectPath: sub}.appliesTo(project))
}

func TestPluginInstallAppliesToSymlinkedProjectPath(t *testing.T) {
	project, sub := projectFixture(t)
	alias := filepath.Join(filepath.Dir(project), "alias")
	if err := os.Symlink(project, alias); err != nil {
		t.Skipf("cannot create a symlink: %v", err)
	}
	for _, cwd := range []string{project, sub} {
		assert.True(t, pluginInstall{Scope: "project", ProjectPath: alias}.appliesTo(cwd), cwd)
	}
}

func TestPluginInstallAppliesToProjectPathInAnotherCase(t *testing.T) {
	project, sub := projectFixture(t)
	upper := filepath.Join(filepath.Dir(project), "PROJECT")
	if _, err := os.Stat(upper); err != nil {
		t.Skip("the temp filesystem is case-sensitive")
	}
	for _, cwd := range []string{project, sub} {
		assert.True(t, pluginInstall{Scope: "project", ProjectPath: upper}.appliesTo(cwd), cwd)
	}
}

func TestDatabricksPluginVersionWithoutReader(t *testing.T) {
	// An agent with no per-agent manifest reader (its format is unverified)
	// reports no version, even when an installed_plugins.json is present, so its
	// manifest is never parsed with another agent's format.
	configDir := t.TempDir()
	writeManifest(t, configDir, `{"plugins":{"databricks@m":[{"version":"0.2.9"}]}}`)
	a := &Agent{
		Name:      "codex",
		ConfigDir: func(_ context.Context) (string, error) { return configDir, nil },
	}
	version, ok := a.DatabricksPluginVersion(t.Context())
	assert.False(t, ok)
	assert.Empty(t, version)
}

// Only a project-scope record for the project that contains cwd is the
// project's own plugin; user and local records are personal.
func TestDatabricksProjectPluginInstallsIn(t *testing.T) {
	project, sub := projectFixture(t)
	other := filepath.Join(filepath.Dir(project), "Other")
	require.NoError(t, os.Mkdir(other, 0o755))
	for _, tc := range []struct {
		install pluginInstall
		want    bool
	}{
		{pluginInstall{Scope: "project", ProjectPath: project}, true},
		{pluginInstall{Scope: "project", ProjectPath: other}, false},
		{pluginInstall{Scope: "local", ProjectPath: project}, false},
		{pluginInstall{Scope: "user"}, false},
	} {
		a := &Agent{pluginInstalls: func(context.Context, *Agent) ([]pluginInstall, bool, error) {
			return []pluginInstall{tc.install}, true, nil
		}}
		got, err := a.DatabricksProjectPluginInstallsIn(t.Context(), sub)
		require.NoError(t, err)
		assert.Equal(t, tc.want, len(got) > 0, "%+v", tc.install)
	}
}

// Each record names the marketplace of its manifest entry, so the agent's own
// CLI can be told which install to remove.
func TestDatabricksPluginInstallsInNamesTheMarketplace(t *testing.T) {
	project, sub := projectFixture(t)
	configDir := t.TempDir()
	writeManifest(t, configDir, `{"plugins":{
		"databricks@databricks-agent-skills":[{"scope":"user"},{"scope":"project","projectPath":`+strconv.Quote(project)+`}],
		"databricks@claude-plugins-official":[{"scope":"user"}],
		"gopls-lsp@claude-plugins-official":[{"scope":"user"}]}}`)
	a := agentWithConfigDir(configDir)

	installs, err := a.DatabricksPluginInstallsIn(t.Context(), "")
	require.NoError(t, err)
	assert.Equal(t, []PluginInstall{
		{Marketplace: "claude-plugins-official", Scope: "user"},
		{Marketplace: "databricks-agent-skills", Scope: "user"},
	}, installs)

	installs, err = a.DatabricksProjectPluginInstallsIn(t.Context(), sub)
	require.NoError(t, err)
	assert.Equal(t, []PluginInstall{{Marketplace: "databricks-agent-skills", Scope: "project"}}, installs)
}

func TestDatabricksPluginConfigured(t *testing.T) {
	codex := filepath.Join(".codex", "config.toml")
	copilot := filepath.Join(".copilot", "config.json")
	codexCache := filepath.Join(".codex", "plugins", "cache")
	copilotCache := filepath.Join(".copilot", "installed-plugins", "databricks-agent-skills", "databricks")
	tests := []struct {
		name  string
		agent string
		file  string
		// body is the file's content, "" for no file; {{home}} stands for
		// the home directory.
		body string
		// cache is a directory to create under the home directory, and
		// cacheFile a file; "" means none.
		cache     string
		cacheFile string
		want      bool
		wantErr   bool
	}{
		{name: "codex without config", agent: NameCodex, file: codex},
		{name: "codex plugin", agent: NameCodex, file: codex, body: "[plugins.\"databricks@databricks-agent-skills\"]\nenabled = true\n", cache: filepath.Join(codexCache, "databricks-agent-skills", "databricks", "0.2.50"), want: true},
		{name: "codex disabled plugin", agent: NameCodex, file: codex, body: "[plugins.\"databricks@other\"]\nenabled = false\n", cache: filepath.Join(codexCache, "other", "databricks", "local"), want: true},
		{name: "codex plugin of a second marketplace", agent: NameCodex, file: codex, body: "[plugins.\"databricks@a\"]\n[plugins.\"databricks@b\"]\n", cache: filepath.Join(codexCache, "b", "databricks", "0.2.50"), want: true},
		// A table left behind without the cache: Codex reports the plugin as
		// not installed and loads nothing.
		{name: "codex table without cache", agent: NameCodex, file: codex, body: "[plugins.\"databricks@databricks-agent-skills\"]\nenabled = true\n"},
		{name: "codex table without version", agent: NameCodex, file: codex, body: "[plugins.\"databricks@databricks-agent-skills\"]\nenabled = true\n", cache: filepath.Join(codexCache, "databricks-agent-skills", "databricks")},
		{name: "codex cache file is not a version", agent: NameCodex, file: codex, body: "[plugins.\"databricks@databricks-agent-skills\"]\nenabled = true\n", cacheFile: filepath.Join(codexCache, "databricks-agent-skills", "databricks", "0.2.50")},
		{name: "codex cache of another marketplace", agent: NameCodex, file: codex, body: "[plugins.\"databricks@databricks-agent-skills\"]\nenabled = true\n", cache: filepath.Join(codexCache, "other", "databricks", "0.2.50")},
		{name: "codex cache without table", agent: NameCodex, file: codex, body: "[marketplaces.databricks-agent-skills]\nsource = \"x\"\n", cache: filepath.Join(codexCache, "databricks-agent-skills", "databricks", "0.2.50")},
		{name: "codex other plugin", agent: NameCodex, file: codex, body: "[plugins.\"databricks-foo@m\"]\nenabled = true\n", cache: filepath.Join(codexCache, "m", "databricks-foo", "0.1.0")},
		{name: "codex unreadable", agent: NameCodex, file: codex, body: "[plugins\n", wantErr: true},
		{name: "copilot without config", agent: NameCopilot, file: copilot},
		{name: "copilot plugin", agent: NameCopilot, file: copilot, body: "// managed\n{\"installedPlugins\": [{\"name\": \"databricks\", \"marketplace\": \"databricks-agent-skills\", \"cache_path\": \"{{home}}/.copilot/installed-plugins/databricks-agent-skills/databricks\"}]}", cache: copilotCache, want: true},
		{name: "copilot plugin of a second marketplace", agent: NameCopilot, file: copilot, body: "{\"installedPlugins\": [{\"name\": \"databricks\", \"cache_path\": \"{{home}}/a\"}, {\"name\": \"databricks\", \"cache_path\": \"{{home}}/.copilot/installed-plugins/databricks-agent-skills/databricks\"}]}", cache: copilotCache, want: true},
		// An entry whose cache directory is missing: Copilot loads nothing
		// (github/copilot-cli#4039).
		{name: "copilot entry without cache", agent: NameCopilot, file: copilot, body: "{\"installedPlugins\": [{\"name\": \"databricks\", \"cache_path\": \"{{home}}/.copilot/installed-plugins/databricks-agent-skills/databricks\"}]}"},
		{name: "copilot entry without cache_path", agent: NameCopilot, file: copilot, body: "{\"installedPlugins\": [{\"name\": \"databricks\"}]}", cache: copilotCache},
		{name: "copilot cache_path is a file", agent: NameCopilot, file: copilot, body: "{\"installedPlugins\": [{\"name\": \"databricks\", \"cache_path\": \"{{home}}/.copilot/installed-plugins/databricks-agent-skills/databricks\"}]}", cacheFile: copilotCache},
		{name: "copilot uninstalled", agent: NameCopilot, file: copilot, body: "// managed\n{\"installedPlugins\": []}"},
		{name: "copilot other plugin", agent: NameCopilot, file: copilot, body: "{\"installedPlugins\": [{\"name\": \"databricks-foo\"}]}"},
		{name: "copilot unreadable", agent: NameCopilot, file: copilot, body: "{not json", wantErr: true},
		{name: "claude has no config reader", agent: NameClaudeCode, file: filepath.Join(".claude", "plugins", pluginManifestFile), body: `{"plugins":{"databricks@m":[{"scope":"user"}]}}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			for _, v := range []string{codexHomeEnv, copilotHomeEnv, claudeHomeEnv} {
				t.Setenv(v, "")
			}
			ctx := env.WithUserHomeDir(t.Context(), home)
			if tt.body != "" {
				path := filepath.Join(home, tt.file)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				body := strings.ReplaceAll(tt.body, "{{home}}", filepath.ToSlash(home))
				require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
			}
			if tt.cache != "" {
				require.NoError(t, os.MkdirAll(filepath.Join(home, tt.cache), 0o755))
			}
			if tt.cacheFile != "" {
				path := filepath.Join(home, tt.cacheFile)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, nil, 0o600))
			}
			got, err := ByName(tt.agent).DatabricksPluginConfigured(ctx)
			assert.Equal(t, tt.wantErr, err != nil, "error: %v", err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDatabricksPluginConfiguredHonorsAgentHomes(t *testing.T) {
	codexHome := t.TempDir()
	copilotHome := t.TempDir()
	t.Setenv(codexHomeEnv, codexHome)
	t.Setenv(copilotHomeEnv, copilotHome)
	ctx := env.WithUserHomeDir(t.Context(), t.TempDir())
	require.NoError(t, os.WriteFile(filepath.Join(codexHome, "config.toml"), []byte("[plugins.\"databricks@m\"]\n"), 0o600))
	require.NoError(t, os.MkdirAll(filepath.Join(codexHome, "plugins", "cache", "m", "databricks", "0.2.50"), 0o755))
	copilotCache := filepath.Join(copilotHome, "installed-plugins", "m", "databricks")
	require.NoError(t, os.MkdirAll(copilotCache, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(copilotHome, "config.json"), []byte(`{"installedPlugins":[{"name":"databricks","cache_path":"`+filepath.ToSlash(copilotCache)+`"}]}`), 0o600))

	for _, name := range []string{NameCodex, NameCopilot} {
		got, err := ByName(name).DatabricksPluginConfigured(ctx)
		require.NoError(t, err)
		assert.True(t, got, name)
	}
}

// Each databricks plugin the config records names its marketplace, and a
// record that cannot be checked does not hide the others. A NUL byte makes
// the record's path unreadable on every platform.
func TestDatabricksPluginConfigMarketplaces(t *testing.T) {
	tests := []struct {
		name    string
		agent   string
		file    string
		body    string
		caches  []string
		want    []string
		wantErr bool
	}{
		{
			name:   "codex",
			agent:  NameCodex,
			file:   filepath.Join(".codex", "config.toml"),
			body:   "[plugins.\"databricks@b\"]\n[plugins.\"databricks@a\"]\n[plugins.\"databricks@c\"]\n",
			caches: []string{filepath.Join(".codex", "plugins", "cache", "a", "databricks", "0.2.50"), filepath.Join(".codex", "plugins", "cache", "b", "databricks", "0.2.50")},
			want:   []string{"a", "b"},
		},
		{
			name:   "codex unreadable cache first",
			agent:  NameCodex,
			file:   filepath.Join(".codex", "config.toml"),
			body:   "[plugins.\"databricks@a\\u0000\"]\n[plugins.\"databricks@b\"]\n",
			caches: []string{filepath.Join(".codex", "plugins", "cache", "b", "databricks", "0.2.50")},
			want:   []string{"b"},
		},
		{
			name:    "codex only an unreadable cache",
			agent:   NameCodex,
			file:    filepath.Join(".codex", "config.toml"),
			body:    "[plugins.\"databricks@a\\u0000\"]\n",
			wantErr: true,
		},
		{
			name:   "copilot",
			agent:  NameCopilot,
			file:   filepath.Join(".copilot", "config.json"),
			body:   `{"installedPlugins": [{"name": "databricks", "marketplace": "a", "cache_path": "{{home}}/a"}, {"name": "databricks", "marketplace": "b", "cache_path": "{{home}}/b"}]}`,
			caches: []string{"a", "b"},
			want:   []string{"a", "b"},
		},
		{
			name:   "copilot unreadable cache first",
			agent:  NameCopilot,
			file:   filepath.Join(".copilot", "config.json"),
			body:   `{"installedPlugins": [{"name": "databricks", "marketplace": "a", "cache_path": "{{home}}/a\u0000"}, {"name": "databricks", "marketplace": "b", "cache_path": "{{home}}/b"}]}`,
			caches: []string{"b"},
			want:   []string{"b"},
		},
		{
			name:    "copilot only an unreadable cache",
			agent:   NameCopilot,
			file:    filepath.Join(".copilot", "config.json"),
			body:    `{"installedPlugins": [{"name": "databricks", "marketplace": "a", "cache_path": "{{home}}/a\u0000"}]}`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			home := t.TempDir()
			for _, v := range []string{codexHomeEnv, copilotHomeEnv} {
				t.Setenv(v, "")
			}
			ctx := env.WithUserHomeDir(t.Context(), home)
			path := filepath.Join(home, tt.file)
			require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
			body := strings.ReplaceAll(tt.body, "{{home}}", filepath.ToSlash(home))
			require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
			for _, cache := range tt.caches {
				require.NoError(t, os.MkdirAll(filepath.Join(home, cache), 0o755))
			}
			got, err := ByName(tt.agent).DatabricksPluginConfigMarketplaces(ctx)
			assert.Equal(t, tt.wantErr, err != nil, "error: %v", err)
			assert.Equal(t, tt.want, got)
		})
	}
}
