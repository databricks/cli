package installer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInstalledTools exercises the real agent registry: claude-code reads its
// authoritative manifest (installed_plugins.json), while agents without a
// verified manifest reader (e.g. codex, copilot) fall back to the CLI install
// state. HOME is redirected to a temp dir so both signals resolve there.
func TestInstalledTools(t *testing.T) {
	tests := []struct {
		name string
		// claudeManifestVersion, when set, is written to claude-code's manifest.
		claudeManifestVersion string
		// globalPlugins / projectPlugins map agent -> recorded state version.
		globalPlugins  map[string]string
		projectPlugins map[string]string
		want           []string
	}{
		{
			name: "none installed",
			want: nil,
		},
		{
			// Direct-from-marketplace install: claude-code version from manifest.
			name:                  "claude manifest version",
			claudeManifestVersion: "0.2.9",
			want:                  []string{"claude-code_0.2.9"},
		},
		{
			// codex has no verified manifest reader, so it uses the state version.
			name:          "state fallback for codex",
			globalPlugins: map[string]string{"codex": "0.2.7"},
			want:          []string{"codex_0.2.7"},
		},
		{
			// Manifest wins over state for claude-code when both are present.
			name:                  "manifest overrides state",
			claudeManifestVersion: "0.2.9",
			globalPlugins:         map[string]string{"claude-code": "0.2.5"},
			want:                  []string{"claude-code_0.2.9"},
		},
		{
			// Project scope wins over global in the state fallback.
			name:           "project state overrides global",
			globalPlugins:  map[string]string{"codex": "0.2.5"},
			projectPlugins: map[string]string{"codex": "0.2.7"},
			want:           []string{"codex_0.2.7"},
		},
		{
			// Manifest (claude) and state (codex) contribute; output is sorted.
			name:                  "mixed sources sorted",
			claudeManifestVersion: "0.2.9",
			globalPlugins:         map[string]string{"codex": "0.2.8"},
			want:                  []string{"claude-code_0.2.9", "codex_0.2.8"},
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
			savePluginState(t, ProjectSkillsDir, tc.projectPlugins)

			assert.Equal(t, tc.want, InstalledTools(t.Context()))
		})
	}
}

// writeClaudeManifest writes a Claude installed_plugins.json recording the
// databricks plugin at the given version under configDir.
func writeClaudeManifest(t *testing.T, configDir, version string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Join(configDir, "plugins"), 0o755))
	body := `{"version":2,"plugins":{"databricks@claude-plugins-official":[{"version":"` + version + `"}]}}`
	require.NoError(t, os.WriteFile(filepath.Join(configDir, "plugins", "installed_plugins.json"), []byte(body), 0o644))
}

// savePluginState writes an install state recording a databricks plugin version
// for each agent into the directory returned by dirFn. It is a no-op when the
// map is empty, so a scope with no plugin installs leaves no state file.
func savePluginState(t *testing.T, dirFn func(context.Context) (string, error), versions map[string]string) {
	t.Helper()
	if len(versions) == 0 {
		return
	}
	plugins := make(map[string]PluginRecord, len(versions))
	for name, v := range versions {
		plugins[name] = PluginRecord{Plugin: "databricks", Version: v}
	}
	dir, err := dirFn(t.Context())
	require.NoError(t, err)
	require.NoError(t, SaveState(dir, &InstallState{
		SchemaVersion: schemaVersionV2,
		Plugins:       plugins,
	}))
}
