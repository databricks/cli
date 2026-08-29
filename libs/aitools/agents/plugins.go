package agents

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/semver"
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
// in the agent's own plugin manifest and whether the plugin is installed at all.
// This catches installs done through `databricks aitools install` and a direct
// `<agent> plugin install`, unlike the CLI's own state file, which only records
// CLI-driven installs. Each agent's manifest format differs, so parsing is
// delegated to a per-agent reader; agents without a verified reader report
// ("", false), i.e. "not installed as far as we can tell".
func (a *Agent) DatabricksPluginVersion(ctx context.Context) (string, bool) {
	if a.pluginVersion == nil {
		return "", false
	}
	return a.pluginVersion(ctx, a)
}

// claudePluginVersion reads the databricks plugin version from Claude Code's
// installed_plugins.json (<ConfigDir>/plugins/), which keys each entry as
// "<id>@<marketplace>" (e.g. "databricks@claude-plugins-official") with one
// record per install scope. Any read/parse failure reports ("", false), so an
// unreadable manifest reads as "not installed". When the plugin is recorded for
// more than one scope, the highest recorded version is returned (they match in
// the common single-scope case, but the cache can retain a stale scope); the
// version is "" when installed but unversioned.
func claudePluginVersion(ctx context.Context, a *Agent) (string, bool) {
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
		// Installed. Report the highest version across scopes;
		best := ""
		for _, install := range installs {
			// Compare as semver (versions are unprefixed, e.g. "0.2.9").
			if install.Version != "" && (best == "" || semver.Compare("v"+install.Version, "v"+best) > 0) {
				best = install.Version
			}
		}
		return best, true
	}
	return "", false
}
