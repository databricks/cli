package root

import (
	"testing"

	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// recordPlugin marks the agent's plugin installed in the global CLI state, the
// simplest signal that drives InstalledTools for any agent.
func recordPlugin(t *testing.T, name, version string) {
	t.Helper()
	dir, err := installer.GlobalSkillsDir(t.Context())
	require.NoError(t, err)
	require.NoError(t, installer.SaveState(dir, &installer.InstallState{
		SchemaVersion: 2,
		Plugins:       map[string]installer.PluginRecord{name: {Plugin: "databricks", Version: version}},
	}))
}

func TestWithAiToolsInUserAgent(t *testing.T) {
	t.Run("none installed emits nothing", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Setenv("USERPROFILE", tmp)
		t.Chdir(t.TempDir())

		ua := useragent.FromContext(withAiToolsInUserAgent(t.Context()))
		assert.NotContains(t, ua, "aitools/")
	})

	t.Run("installed tool becomes an aitools version pair", func(t *testing.T) {
		tmp := t.TempDir()
		t.Setenv("HOME", tmp)
		t.Setenv("USERPROFILE", tmp)
		t.Chdir(t.TempDir())
		recordPlugin(t, "codex", "0.2.9")

		ua := useragent.FromContext(withAiToolsInUserAgent(t.Context()))
		assert.Contains(t, ua, "aitools/codex_0.2.9")
	})
}
