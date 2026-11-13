package agents

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
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
	home := t.TempDir()
	ctx := env.WithUserHomeDir(t.Context(), home)
	for _, key := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "COPILOT_HOME", "GEMINI_CLI_HOME", "XDG_CONFIG_HOME", "APPDATA"} {
		ctx = env.Set(ctx, key, "") //nolint:fatcontext // each key is set on the same context
	}
	return ctx, home
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
			tctx := ctx
			for k, v := range tc.override {
				tctx = env.Set(tctx, k, v) //nolint:fatcontext // each override is set on the same context
			}
			got, err := ByName(tc.agent).MCP.Path(tctx)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestOpenCodeMCPPathPrefersJSONC checks that OpenCode reads opencode.json or
// opencode.jsonc, and that an existing .jsonc is the file to edit, so a second
// file is not created beside it.
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

	// Antigravity's "disabled" key is optional and defaults to false, so the entry does not write it.
	antigravity := ByName(NameAntigravity).MCP.Entry("/bin/databricks", args, nil)
	assert.Equal(t, map[string]any{"command": "/bin/databricks", "args": args}, antigravity)
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

// TestOpenCodeMCPPathReturnsStatError checks that a config path that cannot be
// inspected is an error, not a missing file. Creating opencode.json beside it
// would leave the user's file unread.
func TestOpenCodeMCPPathReturnsStatError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the blocking file has a different error on Windows")
	}
	ctx, home := clearMCPOverrides(t)
	// A regular file where the OpenCode directory should be makes the lookup fail with ENOTDIR.
	require.NoError(t, os.MkdirAll(filepath.Join(home, ".config"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(home, ".config", "opencode"), []byte("x"), 0o600))

	_, err := ByName(NameOpenCode).MCP.Path(ctx)
	require.Error(t, err)
}

// TestOpenCodeMCPPathRefusesBothFiles checks that the CLI does not pick between
// opencode.json and opencode.jsonc when both exist, and names both files.
func TestOpenCodeMCPPathRefusesBothFiles(t *testing.T) {
	ctx, home := clearMCPOverrides(t)
	dir := filepath.Join(home, ".config", "opencode")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	jsonPath := filepath.ToSlash(filepath.Join(dir, "opencode.json"))
	jsoncPath := filepath.ToSlash(filepath.Join(dir, "opencode.jsonc"))
	require.NoError(t, os.WriteFile(jsonPath, []byte("{}"), 0o600))
	require.NoError(t, os.WriteFile(jsoncPath, []byte("{}"), 0o600))

	_, err := ByName(NameOpenCode).MCP.Path(ctx)
	require.Error(t, err)
	assert.ErrorContains(t, err, jsonPath+" and "+jsoncPath)
}

// TestOpenCodeMCPPathDanglingJSONCLink checks that a dangling opencode.jsonc link
// counts as the file, so the write refuses it instead of creating opencode.json.
func TestOpenCodeMCPPathDanglingJSONCLink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("creating symlinks needs privileges on Windows")
	}
	ctx, home := clearMCPOverrides(t)
	dir := filepath.Join(home, ".config", "opencode")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	jsonc := filepath.Join(dir, "opencode.jsonc")
	require.NoError(t, os.Symlink(filepath.Join(dir, "missing.jsonc"), jsonc))

	path, err := ByName(NameOpenCode).MCP.Path(ctx)
	require.NoError(t, err)
	assert.Equal(t, jsonc, path)
}

// TestInstalledForMCP covers the signals that make an agent installed for MCP servers, with the
// agent's own paths under a temporary home and no agent binary on PATH unless a case says so.
func TestInstalledForMCP(t *testing.T) {
	ctx := t.Context()
	setup := func(t *testing.T) string {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		for _, name := range []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "COPILOT_HOME", "GEMINI_CLI_HOME", "XDG_CONFIG_HOME", "APPDATA"} {
			t.Setenv(name, "")
		}
		stubLookPath(t)
		return home
	}
	installed := func(t *testing.T, name string) bool {
		t.Helper()
		a := ByName(name)
		require.NotNil(t, a, name)
		got, err := a.InstalledForMCP(ctx)
		require.NoError(t, err)
		return got
	}

	t.Run("nothing present is not installed", func(t *testing.T) {
		setup(t)
		for _, a := range MCPAgents() {
			assert.False(t, installed(t, a.Name), a.Name)
		}
	})

	t.Run("config dir is installed", func(t *testing.T) {
		home := setup(t)
		require.NoError(t, os.MkdirAll(filepath.Join(home, ".cursor"), 0o755))
		assert.True(t, installed(t, NameCursor))
	})

	t.Run("CLI on PATH alone is installed", func(t *testing.T) {
		setup(t)
		stubLookPath(t, "codex")
		assert.True(t, installed(t, NameCodex))
		assert.False(t, installed(t, NameCopilot))
	})

	t.Run("MCP config file alone is installed", func(t *testing.T) {
		home := setup(t)
		require.NoError(t, os.WriteFile(filepath.Join(home, ".claude.json"), []byte("{}"), 0o600))
		assert.True(t, installed(t, NameClaudeCode))
	})

	t.Run("CODEX_HOME that exists counts for Codex", func(t *testing.T) {
		setup(t)
		codexHome := t.TempDir()
		t.Setenv("CODEX_HOME", codexHome)
		assert.True(t, installed(t, NameCodex))

		t.Setenv("CODEX_HOME", filepath.Join(t.TempDir(), "missing"))
		assert.False(t, installed(t, NameCodex))
	})

	t.Run("CODEX_HOME that is missing is not installed although ~/.codex exists", func(t *testing.T) {
		home := setup(t)
		require.NoError(t, os.MkdirAll(filepath.Join(home, ".codex"), 0o755))
		t.Setenv("CODEX_HOME", filepath.Join(home, "codex-home"))
		assert.False(t, installed(t, NameCodex))
		assert.NoDirExists(t, filepath.Join(home, "codex-home"))
	})

	t.Run("COPILOT_HOME that exists counts for Copilot", func(t *testing.T) {
		setup(t)
		t.Setenv("COPILOT_HOME", t.TempDir())
		assert.True(t, installed(t, NameCopilot))
	})

	t.Run("Gemini is not installed because Antigravity made ~/.gemini", func(t *testing.T) {
		home := setup(t)
		require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini", "antigravity"), 0o755))
		assert.False(t, installed(t, NameGemini))
		assert.True(t, installed(t, NameAntigravity))

		require.NoError(t, os.WriteFile(filepath.Join(home, ".gemini", "projects.json"), []byte("{}"), 0o644))
		assert.True(t, installed(t, NameGemini))
	})

	t.Run("Antigravity is installed from its own directory, without ~/.gemini/config", func(t *testing.T) {
		home := setup(t)
		require.NoError(t, os.MkdirAll(filepath.Join(home, ".gemini", "antigravity"), 0o755))
		assert.NoDirExists(t, filepath.Join(home, ".gemini", "config"))
		assert.True(t, installed(t, NameAntigravity))
	})
}
