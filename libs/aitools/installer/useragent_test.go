package installer

import (
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
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
