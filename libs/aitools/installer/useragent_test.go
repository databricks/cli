package installer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/libs/aitools/agents/agenttest"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWithAiToolsInUserAgent is an end-to-end test of the top-level call tree:
// it drives WithAiToolsInUserAgent against a prepared environment (HOME, plugin
// manifests, and CLI install state) and asserts the resulting user-agent string.
func TestWithAiToolsInUserAgent(t *testing.T) {
	tests := []struct {
		name string
		// claudeManifestVersion, when set, is written to claude-code's manifest.
		claudeManifestVersion string
		// globalPlugins maps agent -> version recorded in the CLI install state.
		globalPlugins  map[string]string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "nothing installed adds no pair",
			wantNotContain: []string{"aitools/"},
		},
		{
			name:                  "version from the tool's own manifest",
			claudeManifestVersion: "0.2.9",
			wantContains:          []string{"aitools/claude-code_0.2.9"},
		},
		{
			name:           "version from the CLI install state",
			globalPlugins:  map[string]string{"codex": "0.2.7"},
			wantContains:   []string{"aitools/codex_0.2.7"},
			wantNotContain: []string{"aitools/claude-code"},
		},
		{
			name:                  "a pair per installed tool",
			claudeManifestVersion: "0.2.9",
			globalPlugins:         map[string]string{"codex": "0.2.8"},
			wantContains:          []string{"aitools/claude-code_0.2.9", "aitools/codex_0.2.8"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			t.Setenv("HOME", tmp)
			t.Setenv("USERPROFILE", tmp)
			// Fresh cwd so project-scope state does not leak from the package dir.
			t.Chdir(t.TempDir())

			if tc.claudeManifestVersion != "" {
				writeClaudeManifest(t, filepath.Join(tmp, ".claude"), tc.claudeManifestVersion)
			}
			savePluginState(t, GlobalSkillsDir, tc.globalPlugins)

			ua := useragent.FromContext(WithAiToolsInUserAgent(t.Context()))
			for _, want := range tc.wantContains {
				assert.Contains(t, ua, want)
			}
			for _, notWant := range tc.wantNotContain {
				assert.NotContains(t, ua, notWant)
			}
		})
	}
}

// TestWithAiDevKitInUserAgent drives WithAiDevKitInUserAgent against a prepared
// HOME containing (or lacking) an ai-dev-kit version marker and asserts the
// resulting user-agent string.
func TestWithAiDevKitInUserAgent(t *testing.T) {
	tests := []struct {
		name string
		// marker, when set, is written to ~/.ai-dev-kit/version.
		marker         string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:           "not installed adds no pair",
			wantNotContain: []string{"aidevkit/"},
		},
		{
			name:         "version from the marker",
			marker:       "1.2.3\n",
			wantContains: []string{"aidevkit/1.2.3"},
		},
		{
			name:         "installed but unversioned reports unknown",
			marker:       "\n",
			wantContains: []string{"aidevkit/unknown"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			// Fresh cwd so a project-scope marker never leaks from the package dir.
			t.Chdir(t.TempDir())

			if tc.marker != "" {
				writeAiDevKitMarker(t, home, tc.marker)
			}

			ua := useragent.FromContext(WithAiDevKitInUserAgent(t.Context()))
			for _, want := range tc.wantContains {
				assert.Contains(t, ua, want)
			}
			for _, notWant := range tc.wantNotContain {
				assert.NotContains(t, ua, notWant)
			}
		})
	}
}

// TestWithCallingAgentToolingInUserAgent checks that the segment reflects only
// the calling agent's own tooling: a plugin installed for Claude Code does not
// make Codex count as installed.
func TestWithCallingAgentToolingInUserAgent(t *testing.T) {
	tests := []struct {
		name     string
		agentEnv []string
		claude   string // claude-code manifest version, when set
		plugins  map[string]string
		state    string // raw global install state, when set
		skill    string // skill dir to create, relative to HOME, when set
		want     string // "" means no aitools-agent segment
	}{
		{name: "no calling agent", claude: "0.2.9", want: ""},
		{name: "agent the CLI cannot install into", agentEnv: []string{"WINDSURF_AGENT", "1"}, want: ""},
		{name: "free-text agent", agentEnv: []string{"AGENT", "my-tool"}, want: ""},
		{name: "claude with its plugin", agentEnv: []string{"CLAUDECODE", "1"}, claude: "0.2.9", want: "aitools-agent/plugin"},
		{name: "codex without tooling while claude has it", agentEnv: []string{"CODEX_CI", "1"}, claude: "0.2.9", want: "aitools-agent/none"},
		{name: "codex with its recorded plugin", agentEnv: []string{"CODEX_CI", "1"}, plugins: map[string]string{"codex": "0.2.8"}, want: "aitools-agent/plugin"},
		{name: "codex while the CLI state records only claude's plugin", agentEnv: []string{"CODEX_CI", "1"}, plugins: map[string]string{"claude-code": "0.2.8"}, want: "aitools-agent/none"},
		{name: "codex with a corrupt CLI state", agentEnv: []string{"CODEX_CI", "1"}, state: "{not json", want: "aitools-agent/none"},
		{name: "claude with only a stale CLI record", agentEnv: []string{"CLAUDECODE", "1"}, plugins: map[string]string{"claude-code": "0.2.8"}, want: "aitools-agent/none"},
		{name: "claude with only skills", agentEnv: []string{"CLAUDECODE", "1"}, skill: filepath.Join(".claude", "skills", "databricks-apps"), want: "aitools-agent/skills"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// The first t.TempDir picks the temp root, so create both before
			// SetAgentEnv clears TMPDIR / TEMP.
			tmp := t.TempDir()
			cwd := t.TempDir()
			agenttest.SetAgentEnv(t, tc.agentEnv...)
			t.Setenv("HOME", tmp)
			t.Setenv("USERPROFILE", tmp)
			// A .git in cwd keeps the project lookup out of the host's directories.
			require.NoError(t, os.Mkdir(filepath.Join(cwd, ".git"), 0o755))
			t.Chdir(cwd)
			if tc.claude != "" {
				writeClaudeManifest(t, filepath.Join(tmp, ".claude"), tc.claude)
			}
			savePluginState(t, GlobalSkillsDir, tc.plugins)
			if tc.state != "" {
				dir, err := GlobalSkillsDir(t.Context())
				require.NoError(t, err)
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(dir, stateFileName), []byte(tc.state), 0o600))
			}
			if tc.skill != "" {
				require.NoError(t, os.MkdirAll(filepath.Join(tmp, tc.skill), 0o755))
			}

			ua := useragent.FromContext(WithCallingAgentToolingInUserAgent(t.Context()))
			if tc.want == "" {
				assert.NotContains(t, ua, "aitools-agent/")
				return
			}
			assert.Contains(t, ua, tc.want)
		})
	}
}
