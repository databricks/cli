package agents

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// pluginManifestFile is the file an agent's own plugin CLI writes to record
// installed plugins. Claude Code stores it at <configDir>/plugins and keys each
// entry as "<id>@<marketplace>" (e.g. "databricks@claude-plugins-official").
const pluginManifestFile = "installed_plugins.json"

// pluginInstall is one install record in an agent's plugin manifest. A plugin
// installed in more than one scope has several records; only the version is
// consumed here.
type pluginInstall struct {
	Version string `json:"version"`
}

// DatabricksPluginVersion reports the recorded version of the databricks plugin
// in the agent's own plugin manifest (<ConfigDir>/plugins/installed_plugins.json,
// Claude Code's format) and whether the plugin is installed at all. It matches
// an "<id>@<marketplace>" key whose id is the databricks plugin. This catches
// installs done through `databricks aitools install` and a direct `<agent>
// plugin install`, unlike the CLI's own state file, which only records
// CLI-driven installs. Any read/parse failure reports ("", false), so an
// unreadable manifest reads as "not installed". When the plugin is recorded for
// more than one scope, the first recorded version is returned (they match in the
// common single-scope case); the version is "" when installed but unversioned.
func (a *Agent) DatabricksPluginVersion(ctx context.Context) (string, bool) {
	configDir, err := a.ConfigDir(ctx)
	if err != nil {
		return "", false
	}
	data, err := os.ReadFile(filepath.Join(configDir, "plugins", pluginManifestFile))
	if err != nil {
		return "", false
	}
	var manifest struct {
		Plugins map[string][]pluginInstall `json:"plugins"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return "", false
	}
	for key, installs := range manifest.Plugins {
		id, _, _ := strings.Cut(key, "@")
		if id != databricksPluginID {
			continue
		}
		if len(installs) > 0 {
			return installs[0].Version, true
		}
		return "", true
	}
	return "", false
}
