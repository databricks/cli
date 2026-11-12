package agents

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeConfigDir(t *testing.T) {
	home := t.TempDir()
	custom := t.TempDir()
	for _, tc := range []struct {
		name     string
		override string
		want     string
	}{
		{"default", "", filepath.Join(home, ".claude")},
		{"custom", custom, custom},
		{"relative", "custom-claude", "custom-claude"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := env.WithUserHomeDir(t.Context(), home)
			ctx = env.Set(ctx, "CLAUDE_CONFIG_DIR", tc.override)
			agent := ByName(NameClaudeCode)
			dir, err := agent.ConfigDir(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, dir)
			skills, err := agent.SkillsDir(ctx)
			require.NoError(t, err)
			assert.Equal(t, filepath.Join(tc.want, "skills"), skills)
		})
	}
}

func TestSupportedNamesMatchesRegistry(t *testing.T) {
	names := SupportedNames()
	assert.Len(t, names, len(Registry))
	for i, a := range Registry {
		assert.Equal(t, a.DisplayName, names[i])
	}
}

func TestMCPAgents(t *testing.T) {
	var names []string
	for _, a := range MCPAgents() {
		names = append(names, a.Name)
	}
	assert.Equal(t, []string{
		NameClaudeCode, NameCursor, NameCodex, NameOpenCode, NameCopilot, NameAntigravity, NameGemini,
	}, names)
	for _, name := range []string{NamePi, NameGoose, NameKiro} {
		assert.False(t, ByName(name).SupportsMCP(), name)
	}
}

// clearMCPOverrides returns ctx with a home dir and no override variables, so
// the path tests do not depend on the developer's shell.
func clearMCPOverrides(t *testing.T) (context.Context, string) {
	for _, key := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "COPILOT_HOME", "GEMINI_CLI_HOME", "XDG_CONFIG_HOME", "APPDATA"} {
		t.Setenv(key, "")
	}
	home := t.TempDir()
	return env.WithUserHomeDir(t.Context(), home), home
}

func TestMCPPaths(t *testing.T) {
	ctx, home := clearMCPOverrides(t)
	custom := t.TempDir()
	for _, tc := range []struct {
		name     string
		agent    string
		override map[string]string
		want     string
	}{
		{"claude default", NameClaudeCode, nil, filepath.Join(home, ".claude.json")},
		{"claude config dir", NameClaudeCode, map[string]string{"CLAUDE_CONFIG_DIR": custom}, filepath.Join(custom, ".claude.json")},
		{"cursor", NameCursor, nil, filepath.Join(home, ".cursor", "mcp.json")},
		{"codex default", NameCodex, nil, filepath.Join(home, ".codex", "config.toml")},
		{"codex home", NameCodex, map[string]string{"CODEX_HOME": custom}, filepath.Join(custom, "config.toml")},
		{"opencode default", NameOpenCode, nil, filepath.Join(home, ".config", "opencode", "opencode.json")},
		{"opencode xdg", NameOpenCode, map[string]string{"XDG_CONFIG_HOME": custom}, filepath.Join(custom, "opencode", "opencode.json")},
		{"copilot default", NameCopilot, nil, filepath.Join(home, ".copilot", "mcp-config.json")},
		{"copilot home", NameCopilot, map[string]string{"COPILOT_HOME": custom}, filepath.Join(custom, "mcp-config.json")},
		{"gemini default", NameGemini, nil, filepath.Join(home, ".gemini", "settings.json")},
		{"gemini cli home", NameGemini, map[string]string{"GEMINI_CLI_HOME": custom}, filepath.Join(custom, ".gemini", "settings.json")},
		// The MCP file is the one from the Antigravity docs, not the skills directory.
		{"antigravity", NameAntigravity, nil, filepath.Join(home, ".gemini", "config", "mcp_config.json")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.override {
				t.Setenv(k, v)
			}
			got, err := ByName(tc.agent).MCP.Path(ctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// OpenCode reads opencode.json or opencode.jsonc. An existing .jsonc is the
// file to edit, so a second file is not created beside it.
func TestOpenCodeMCPPathPrefersJSONC(t *testing.T) {
	ctx, home := clearMCPOverrides(t)
	dir := filepath.Join(home, ".config", "opencode")
	path, err := ByName(NameOpenCode).MCP.Path(ctx)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "opencode.json"), path)

	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "opencode.jsonc"), []byte("{}"), 0o600))
	path, err = ByName(NameOpenCode).MCP.Path(ctx)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "opencode.jsonc"), path)
}

func TestMCPEntries(t *testing.T) {
	args := []string{"aitools", "mcp", "serve"}
	pinned := map[string]string{"DATABRICKS_CONFIG_FILE": "/home/u/.databrickscfg"}

	claude := ByName(NameClaudeCode).MCP.Entry("/bin/databricks", args, nil)
	assert.Equal(t, map[string]any{"type": "stdio", "command": "/bin/databricks", "args": args, "env": map[string]string{}}, claude)

	codex := ByName(NameCodex).MCP.Entry("/bin/databricks", args, nil)
	assert.Equal(t, map[string]any{"command": "/bin/databricks", "args": args}, codex, "env is omitted when empty")
	assert.Equal(t, map[string]any{"command": "/bin/databricks", "args": args, "env": pinned},
		ByName(NameCodex).MCP.Entry("/bin/databricks", args, pinned))

	copilot := ByName(NameCopilot).MCP.Entry("/bin/databricks", args, nil)
	assert.Equal(t, "local", copilot["type"])
	assert.Equal(t, []string{"*"}, copilot["tools"])

	// OpenCode uses one command array, the "environment" key and no "enabled" key.
	opencode := ByName(NameOpenCode).MCP.Entry("/bin/databricks", args, pinned)
	assert.Equal(t, map[string]any{
		"type":        "local",
		"command":     []string{"/bin/databricks", "aitools", "mcp", "serve"},
		"environment": pinned,
	}, opencode)

	antigravity := ByName(NameAntigravity).MCP.Entry("/bin/databricks", args, nil)
	assert.Equal(t, false, antigravity["disabled"])
}

func TestSkillsOnlyNamesMatchesRegistry(t *testing.T) {
	names := SkillsOnlyNames()
	// Skills-only agents (Plugin nil) are listed; plugin agents are not.
	assert.Contains(t, names, "Pi")
	assert.Contains(t, names, "Gemini CLI")
	assert.Contains(t, names, "Goose")
	assert.Contains(t, names, "Kiro")
	assert.NotContains(t, names, "Claude Code")
	for _, a := range Registry {
		if a.Plugin != nil {
			assert.NotContains(t, names, a.DisplayName)
		}
	}
}
