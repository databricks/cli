package agents

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSkillAgentRegistryPaths(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	ctx = env.Set(ctx, "XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	ctx = env.Set(ctx, "APPDATA", "")

	gooseGlobalDir := filepath.Join(home, ".config", "goose", "skills")
	if runtime.GOOS == "windows" {
		gooseGlobalDir = filepath.Join(home, "AppData", "Roaming", "Block", "goose", "config", "skills")
	}

	tests := []struct {
		name        string
		binary      string
		displayName string
		globalDir   string
		projectDir  string
	}{
		{NamePi, "pi", "Pi", filepath.Join(home, ".pi", "agent", "skills"), filepath.Join(cwd, ".pi", "skills")},
		{NameGemini, "gemini", "Gemini CLI", filepath.Join(home, ".gemini", "skills"), filepath.Join(cwd, ".gemini", "skills")},
		{NameGoose, "goose", "Goose", gooseGlobalDir, filepath.Join(cwd, ".goose", "skills")},
		{NameKiro, "kiro", "Kiro", filepath.Join(home, ".kiro", "skills"), filepath.Join(cwd, ".kiro", "skills")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := ByName(tc.name)
			require.NotNil(t, a)
			assert.Equal(t, tc.binary, a.Binary)
			assert.Equal(t, tc.displayName, a.DisplayName)
			assert.True(t, a.SupportsProjectScope)
			assert.Nil(t, a.Plugin)

			globalDir, err := a.SkillsDir(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.globalDir, globalDir)
			assert.Equal(t, tc.projectDir, a.ProjectSkillsDir(cwd))
		})
	}
}

func TestDetectProjectInstalled(t *testing.T) {
	cwd := t.TempDir()
	for _, name := range []string{NamePi, NameGemini, NameGoose, NameKiro} {
		dir := filepath.Join(ByName(name).ProjectSkillsDir(cwd), "databricks-core")
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	// A non-databricks skill must not count as a Databricks install.
	require.NoError(t, os.MkdirAll(filepath.Join(cwd, ".pi", "skills", "other-skill"), 0o755))

	var names []string
	for _, a := range DetectProjectInstalled(cwd) {
		names = append(names, a.Name)
	}
	assert.ElementsMatch(t, []string{NamePi, NameGemini, NameGoose, NameKiro}, names)
}

// TestRegistryContract pins what each registry agent promises users: the binary
// probed on PATH, the plugin installed for it, where its skills are written and
// whether it supports project scope. Adding, removing or changing an agent must
// update this table, so an accidental edit to the registry fails here.
func TestRegistryContract(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	// Clear the per-agent overrides so every directory derives from home.
	for _, v := range []string{"CLAUDE_CONFIG_DIR", "PI_CODING_AGENT_DIR", "GEMINI_CLI_HOME", "GOOSE_PATH_ROOT", "XDG_CONFIG_HOME", "APPDATA"} {
		t.Setenv(v, "")
	}

	gooseGlobalDir := filepath.Join(home, ".config", "goose", "skills")
	if runtime.GOOS == "windows" {
		gooseGlobalDir = filepath.Join(home, "AppData", "Roaming", "Block", "goose", "config", "skills")
	}
	databricks := &PluginSpec{Marketplace: "databricks-agent-skills", ID: "databricks", Source: "databricks/databricks-agent-skills"}
	claude := &PluginSpec{Marketplace: "claude-plugins-official", ID: "databricks", Source: "anthropics/claude-plugins-official", Shared: true}

	tests := []struct {
		name          string
		displayName   string
		binary        string
		plugin        *PluginSpec
		mandatoryFile string
		globalDir     string
		// projectDir is empty for agents without project scope.
		projectDir string
	}{
		{NameClaudeCode, "Claude Code", "claude", claude, "", filepath.Join(home, ".claude", "skills"), filepath.Join(cwd, ".claude", "skills")},
		{NameCursor, "Cursor", "cursor-agent", nil, "", filepath.Join(home, ".cursor", "skills"), ""},
		{NameCodex, "Codex CLI", "codex", databricks, "", filepath.Join(home, ".codex", "skills"), ""},
		{NameOpenCode, "OpenCode", "opencode", nil, "", filepath.Join(home, ".config", "opencode", "skills"), ""},
		{NameCopilot, "GitHub Copilot", "copilot", databricks, "", filepath.Join(home, ".copilot", "skills"), ""},
		{NameAntigravity, "Antigravity", "", nil, "", filepath.Join(home, ".gemini", "antigravity", "global_skills"), ""},
		{NamePi, "Pi", "pi", nil, "", filepath.Join(home, ".pi", "agent", "skills"), filepath.Join(cwd, ".pi", "skills")},
		{NameGemini, "Gemini CLI", "gemini", nil, "projects.json", filepath.Join(home, ".gemini", "skills"), filepath.Join(cwd, ".gemini", "skills")},
		{NameGoose, "Goose", "goose", nil, "", gooseGlobalDir, filepath.Join(cwd, ".goose", "skills")},
		{NameKiro, "Kiro", "kiro", nil, "", filepath.Join(home, ".kiro", "skills"), filepath.Join(cwd, ".kiro", "skills")},
	}

	var names []string
	for _, a := range Registry {
		names = append(names, a.Name)
	}
	var want []string
	for _, tc := range tests {
		want = append(want, tc.name)
	}
	require.Equal(t, want, names, "registry agents or their order changed; update this table")

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := ByName(tc.name)
			require.NotNil(t, a)
			assert.Equal(t, tc.displayName, a.DisplayName)
			assert.Equal(t, tc.binary, a.Binary)
			assert.Equal(t, tc.plugin, a.Plugin)
			assert.Equal(t, tc.mandatoryFile, a.MandatoryFile)

			globalDir, err := a.SkillsDir(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.globalDir, globalDir)

			assert.Equal(t, tc.projectDir != "", a.SupportsProjectScope)
			if tc.projectDir != "" {
				assert.Equal(t, tc.projectDir, a.ProjectSkillsDir(cwd))
			}
		})
	}
}
