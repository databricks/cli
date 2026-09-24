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

type pluginInstall struct {
	Scope   string `json:"scope"`
	Version string `json:"version"`
}

// DatabricksPluginVersionForScope reports the version from the agent's own
// manifest for nativeScope. An empty nativeScope retains the historical
// highest-version behavior for callers that do not know a native scope.
func (a *Agent) DatabricksPluginVersionForScope(ctx context.Context, nativeScope string) (string, bool) {
	if a.pluginVersion == nil {
		return "", false
	}
	return a.pluginVersion(ctx, a, nativeScope)
}

// HasPluginVersionReader reports whether this agent has a verified manifest
// reader. Callers may fall back to CLI state only when this is false.
func (a *Agent) HasPluginVersionReader() bool {
	return a.pluginVersion != nil
}

// DatabricksPluginVersion is the compatibility wrapper for unscoped lookups.
func (a *Agent) DatabricksPluginVersion(ctx context.Context) (string, bool) {
	return a.DatabricksPluginVersionForScope(ctx, "")
}

func claudePluginVersion(ctx context.Context, a *Agent, nativeScope string) (string, bool) {
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
		if nativeScope != "" {
			for _, install := range installs {
				if install.Scope == nativeScope {
					return install.Version, true
				}
			}
			if len(installs) == 1 && installs[0].Scope == "" {
				return installs[0].Version, true
			}
			return "", false
		}
		best := ""
		for _, install := range installs {
			if install.Version != "" && (best == "" || semver.Compare("v"+install.Version, "v"+best) > 0) {
				best = install.Version
			}
		}
		return best, true
	}
	return "", false
}
