package agents

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// agentWithConfigDir returns an Agent whose ConfigDir resolves to dir.
func agentWithConfigDir(dir string) *Agent {
	return &Agent{
		Name:        "test-agent",
		DisplayName: "Test Agent",
		ConfigDir:   func(_ context.Context) (string, error) { return dir, nil },
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
