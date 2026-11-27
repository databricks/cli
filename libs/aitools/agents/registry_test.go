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

func TestDetectProjectInstalled(t *testing.T) {
	cwd := t.TempDir()
	for _, name := range []string{NamePi, NameGemini, NameGoose, NameKiro, NameCodex} {
		dir := filepath.Join(ByName(name).ProjectSkillsDir(cwd), "databricks-core")
		require.NoError(t, os.MkdirAll(dir, 0o755))
	}
	// A non-databricks skill must not count as a Databricks install.
	require.NoError(t, os.MkdirAll(filepath.Join(cwd, ".pi", "skills", "other-skill"), 0o755))

	var names []string
	for _, a := range DetectProjectInstalled(cwd) {
		names = append(names, a.Name)
	}
	assert.ElementsMatch(t, []string{NamePi, NameGemini, NameGoose, NameKiro, NameCodex}, names)
}

// TestRegistryContract pins each registry agent's name, display name, the
// binary probed on PATH, the plugin installed for it, its marker file, where
// its skills are written and whether its plugin and skills support project scope. Adding,
// removing or changing one of these fields must update this table. The MCP
// spec and plugin manifest readers are not covered here.
func TestRegistryContract(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	// Clear the per-agent overrides so these directories derive from home.
	for _, v := range []string{"CLAUDE_CONFIG_DIR", "PI_CODING_AGENT_DIR", "GEMINI_CLI_HOME", "GOOSE_PATH_ROOT"} {
		t.Setenv(v, "")
	}
	// Point the config roots away from their defaults so the OpenCode and Goose
	// rows fail if either agent stops honoring them.
	xdg := filepath.Join(home, "xdg")
	appData := filepath.Join(home, "appdata")
	t.Setenv("XDG_CONFIG_HOME", xdg)
	t.Setenv("APPDATA", appData)

	openCodeGlobalDir := filepath.Join(xdg, "opencode", "skills")
	gooseGlobalDir := filepath.Join(xdg, "goose", "skills")
	if runtime.GOOS == "windows" {
		openCodeGlobalDir = filepath.Join(appData, "opencode", "skills")
		gooseGlobalDir = filepath.Join(appData, "Block", "goose", "config", "skills")
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
		// projectPlugin is whether the plugin installs in project scope.
		projectPlugin bool
		// projectDir is empty for agents without project-scoped skills.
		projectDir string
	}{
		{"claude-code", "Claude Code", "claude", claude, "", filepath.Join(home, ".claude", "skills"), true, filepath.Join(cwd, ".claude", "skills")},
		{"cursor", "Cursor", "cursor-agent", nil, "", filepath.Join(home, ".cursor", "skills"), false, ""},
		{"codex", "Codex CLI", "codex", databricks, "", filepath.Join(home, ".codex", "skills"), false, filepath.Join(cwd, ".agents", "skills")},
		{"opencode", "OpenCode", "opencode", nil, "", openCodeGlobalDir, false, ""},
		{"copilot", "GitHub Copilot", "copilot", databricks, "", filepath.Join(home, ".copilot", "skills"), false, ""},
		{"antigravity", "Antigravity", "", nil, "", filepath.Join(home, ".gemini", "antigravity", "global_skills"), false, ""},
		{"pi", "Pi", "pi", nil, "", filepath.Join(home, ".pi", "agent", "skills"), false, filepath.Join(cwd, ".pi", "skills")},
		{"gemini", "Gemini CLI", "gemini", nil, "projects.json", filepath.Join(home, ".gemini", "skills"), false, filepath.Join(cwd, ".gemini", "skills")},
		{"goose", "Goose", "goose", nil, "", gooseGlobalDir, false, filepath.Join(cwd, ".goose", "skills")},
		{"kiro", "Kiro", "kiro", nil, "", filepath.Join(home, ".kiro", "skills"), false, filepath.Join(cwd, ".kiro", "skills")},
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

			assert.Equal(t, tc.projectPlugin, a.SupportsProjectPlugin)
			assert.Equal(t, tc.projectDir != "", a.SupportsProjectSkills)
			if tc.projectDir != "" {
				assert.Equal(t, tc.projectDir, a.ProjectSkillsDir(cwd))
			}
		})
	}
}
