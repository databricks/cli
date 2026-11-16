package agents_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallingAgent(t *testing.T) {
	cases := []struct {
		name string
		env  []string
		want string // registry agent name; "" means no calling agent
	}{
		{name: "claude code", env: []string{"CLAUDECODE", "1"}, want: agents.NameClaudeCode},
		{name: "cursor", env: []string{"CURSOR_AGENT", "1"}, want: agents.NameCursor},
		{name: "codex", env: []string{"CODEX_CI", "1"}, want: agents.NameCodex},
		{name: "opencode", env: []string{"OPENCODE", "1"}, want: agents.NameOpenCode},
		{name: "copilot", env: []string{"COPILOT_CLI", "1"}, want: agents.NameCopilot},
		{name: "antigravity", env: []string{"ANTIGRAVITY_AGENT", "1"}, want: agents.NameAntigravity},
		{name: "pi", env: []string{"PI_CODING_AGENT", "1"}, want: agents.NamePi},
		{name: "gemini", env: []string{"GEMINI_CLI", "1"}, want: agents.NameGemini},
		{name: "goose", env: []string{"GOOSE_TERMINAL", "1"}, want: agents.NameGoose},
		{name: "kiro", env: []string{"KIRO", "1"}, want: agents.NameKiro},
		{name: "agent the CLI cannot install into", env: []string{"WINDSURF_AGENT", "1"}},
		{name: "free-text agent", env: []string{"AGENT", "some-internal-tool"}},
		{name: "free-text agent named like a registry agent", env: []string{"AGENT", "copilot"}, want: agents.NameCopilot},
		{name: "no agent"},
		{name: "several agents", env: []string{"CLAUDECODE", "1", "CURSOR_AGENT", "1"}},
	}

	// Every registry agent must be reachable through its SDK env var, or it
	// could never be the calling agent.
	covered := map[string]bool{}
	for _, tc := range cases {
		covered[tc.want] = true
	}
	for _, a := range agents.Registry {
		assert.True(t, covered[a.Name], "no case reaches registry agent %q", a.Name)
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			testutil.SetAgentEnv(t, tc.env...)
			a := agents.CallingAgent()
			if tc.want == "" {
				assert.Nil(t, a)
				return
			}
			require.NotNil(t, a)
			assert.Equal(t, tc.want, a.Name)
		})
	}
}

// toolingFixture isolates HOME, the working directory and agent config env so
// DatabricksTooling reads only what the test writes. cwd holds a .git, so the
// project lookup never walks above it into the host's directories.
func toolingFixture(t *testing.T) (home, cwd string) {
	home = t.TempDir()
	cwd = t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(cwd, ".git"), 0o755))
	t.Chdir(cwd)
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("CLAUDE_CONFIG_DIR", "")
	t.Setenv("PI_CODING_AGENT_DIR", "")
	return home, cwd
}

func mkdir(t *testing.T, parts ...string) {
	require.NoError(t, os.MkdirAll(filepath.Join(parts...), 0o755))
}

func writeFile(t *testing.T, path, body string) {
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
}

// writeClaudeRecord writes a Claude manifest with one databricks plugin record
// in the given scope and project path (the field Claude Code writes for the
// project and local scopes).
func writeClaudeRecord(t *testing.T, home, scope, projectPath string) {
	record := map[string]string{"scope": scope, "version": "0.2.9"}
	if projectPath != "" {
		record["projectPath"] = projectPath
	}
	body, err := json.Marshal(map[string]any{
		"version": 2,
		"plugins": map[string]any{"databricks@claude-plugins-official": []any{record}},
	})
	require.NoError(t, err)
	writeFile(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"), string(body))
}

func TestDatabricksTooling(t *testing.T) {
	cases := []struct {
		name  string
		agent string
		setup func(t *testing.T, home, cwd string)
		// recorded is what pluginRecorded (the CLI install state) reports.
		recorded bool
		want     agents.Tooling
	}{
		{
			name:  "nothing installed",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {},
			want:  agents.ToolingNone,
		},
		{
			name:  "claude plugin in its own manifest",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				writeFile(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
					`{"plugins":{"databricks@claude-plugins-official":[{"version":"0.2.9"}]}}`)
			},
			want: agents.ToolingPlugin,
		},
		{
			name:  "stale claude record in the CLI state, plugin gone from claude's manifest",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				writeFile(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"), `{"plugins":{}}`)
			},
			recorded: true,
			want:     agents.ToolingNone,
		},
		{
			name:     "stale claude record in the CLI state, no claude manifest at all",
			agent:    agents.NameClaudeCode,
			setup:    func(t *testing.T, home, cwd string) {},
			recorded: true,
			want:     agents.ToolingNone,
		},
		{
			name:  "claude project-scope plugin for another project",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				writeClaudeRecord(t, home, "project", filepath.Join(home, "repoA"))
			},
			want: agents.ToolingNone,
		},
		{
			name:  "claude project-scope plugin for a sibling with a shared prefix",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				writeClaudeRecord(t, home, "project", cwd+"x")
			},
			want: agents.ToolingNone,
		},
		{
			name:  "claude project-scope plugin for the working directory",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				writeClaudeRecord(t, home, "project", cwd)
			},
			want: agents.ToolingPlugin,
		},
		{
			name:  "claude local-scope plugin for an ancestor of the working directory",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				writeClaudeRecord(t, home, "local", filepath.Dir(cwd))
			},
			want: agents.ToolingPlugin,
		},
		{
			name:  "claude project-scope record without a project path",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				writeClaudeRecord(t, home, "project", "")
			},
			want: agents.ToolingNone,
		},
		{
			name:  "claude managed-scope plugin",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				writeClaudeRecord(t, home, "managed", "")
			},
			want: agents.ToolingPlugin,
		},
		{
			name:     "codex plugin recorded in the CLI state",
			agent:    agents.NameCodex,
			setup:    func(t *testing.T, home, cwd string) {},
			recorded: true,
			want:     agents.ToolingPlugin,
		},
		{
			name:     "copilot plugin recorded in the CLI state",
			agent:    agents.NameCopilot,
			setup:    func(t *testing.T, home, cwd string) {},
			recorded: true,
			want:     agents.ToolingPlugin,
		},
		{
			name:  "another agent's plugin does not count",
			agent: agents.NameCodex,
			setup: func(t *testing.T, home, cwd string) {
				writeFile(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"),
					`{"plugins":{"databricks@claude-plugins-official":[{"version":"0.2.9"}]}}`)
			},
			want: agents.ToolingNone,
		},
		{
			name:  "cursor skills in its own dir",
			agent: agents.NameCursor,
			setup: func(t *testing.T, home, cwd string) {
				mkdir(t, home, ".cursor", "skills", "databricks-apps")
			},
			want: agents.ToolingSkills,
		},
		{
			name:  "skills only in the shared store do not count",
			agent: agents.NameCursor,
			setup: func(t *testing.T, home, cwd string) {
				mkdir(t, home, ".databricks", "aitools", "skills", "databricks")
			},
			want: agents.ToolingNone,
		},
		{
			name:  "another agent's skills do not count",
			agent: agents.NameCodex,
			setup: func(t *testing.T, home, cwd string) {
				mkdir(t, home, ".cursor", "skills", "databricks")
			},
			want: agents.ToolingNone,
		},
		{
			name:  "non-databricks skills do not count",
			agent: agents.NameKiro,
			setup: func(t *testing.T, home, cwd string) {
				mkdir(t, home, ".kiro", "skills", "other-skill")
			},
			want: agents.ToolingNone,
		},
		{
			name:  "project-scoped skills",
			agent: agents.NameGoose,
			setup: func(t *testing.T, home, cwd string) {
				mkdir(t, cwd, ".goose", "skills", "databricks")
			},
			want: agents.ToolingSkills,
		},
		{
			name:  "project skills at the repository root, CLI in a subdirectory",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				mkdir(t, cwd, ".claude", "skills", "databricks-apps")
				mkdir(t, cwd, "jobs", "src")
				t.Chdir(filepath.Join(cwd, "jobs", "src"))
			},
			want: agents.ToolingSkills,
		},
		{
			name:  "project skills above the repository root do not count",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				mkdir(t, cwd, ".claude", "skills", "databricks-apps")
				// A git worktree's .git is a file; the walk stops there too.
				writeFile(t, filepath.Join(cwd, "repo", ".git"), "gitdir: elsewhere\n")
				mkdir(t, cwd, "repo", "jobs")
				t.Chdir(filepath.Join(cwd, "repo", "jobs"))
			},
			want: agents.ToolingNone,
		},
		{
			name:  "project skills found through a symlinked working directory",
			agent: agents.NameClaudeCode,
			setup: func(t *testing.T, home, cwd string) {
				mkdir(t, cwd, "repo", ".git")
				mkdir(t, cwd, "repo", ".claude", "skills", "databricks-apps")
				mkdir(t, cwd, "repo", "jobs")
				alias := filepath.Join(cwd, "alias")
				if err := os.Symlink(filepath.Join(cwd, "repo", "jobs"), alias); err != nil {
					t.Skipf("cannot create a symlink: %v", err)
				}
				// t.Chdir sets PWD, so os.Getwd reports the alias.
				t.Chdir(alias)
			},
			want: agents.ToolingSkills,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			home, cwd := toolingFixture(t)
			tc.setup(t, home, cwd)
			ctx := env.WithUserHomeDir(t.Context(), home)
			a := agents.ByName(tc.agent)
			require.NotNil(t, a)
			assert.Equal(t, tc.want, a.DatabricksTooling(ctx, func() bool { return tc.recorded }))
		})
	}
}

func TestDatabricksToolingClaudePluginInSeveralMarketplaces(t *testing.T) {
	home, _ := toolingFixture(t)
	// A user-scope install from one marketplace applies everywhere, even next
	// to a project-scope install from another marketplace for another project.
	body, err := json.Marshal(map[string]any{
		"plugins": map[string]any{
			"databricks@claude-plugins-official": []any{map[string]string{"scope": "user", "version": "0.2.9"}},
			"databricks@databricks-agent-skills": []any{map[string]string{"scope": "project", "version": "0.2.9", "projectPath": filepath.Join(home, "repoA")}},
		},
	})
	require.NoError(t, err)
	writeFile(t, filepath.Join(home, ".claude", "plugins", "installed_plugins.json"), string(body))
	ctx := env.WithUserHomeDir(t.Context(), home)
	a := agents.ByName(agents.NameClaudeCode)
	require.NotNil(t, a)
	// TestDatabricksPluginVersionAcrossMarketplaces checks that the records of
	// every marketplace entry are read.
	assert.Equal(t, agents.ToolingPlugin, a.DatabricksTooling(ctx, func() bool { return false }))
}

// The CLI install state is not read for agents whose own manifest is
// authoritative (Claude Code) or that have no plugin (Cursor).
func TestDatabricksToolingSkipsInstallState(t *testing.T) {
	for _, name := range []string{agents.NameClaudeCode, agents.NameCursor} {
		t.Run(name, func(t *testing.T) {
			home, _ := toolingFixture(t)
			ctx := env.WithUserHomeDir(t.Context(), home)
			a := agents.ByName(name)
			require.NotNil(t, a)
			a.DatabricksTooling(ctx, func() bool {
				t.Error("pluginRecorded called")
				return true
			})
		})
	}
}
