package agents

import (
	"context"
	"os"
	"path/filepath"
	"testing"

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
func TestDatabricksProjectPluginIn(t *testing.T) {
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
		got, err := a.DatabricksProjectPluginIn(t.Context(), sub)
		require.NoError(t, err)
		assert.Equal(t, tc.want, got, "%+v", tc.install)
	}
}
