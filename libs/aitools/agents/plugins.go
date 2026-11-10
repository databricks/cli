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
// installed in more than one scope has several records.
type pluginInstall struct {
	Version string `json:"version"`
	// Scope is Claude Code's install scope: "managed", "user", "project" or
	// "local".
	Scope string `json:"scope"`
	// ProjectPath is the project the record applies to. Claude Code requires it
	// for the project and local scopes.
	ProjectPath string `json:"projectPath"`
}

// appliesTo reports whether the agent loads this install when run in cwd:
// always for machine-wide scopes, and for project and local scopes only inside
// the recorded project (cwd is the project or a directory under it). A record
// with an unknown scope counts, so a new machine-wide scope is not missed.
func (p pluginInstall) appliesTo(cwd string) bool {
	switch p.Scope {
	case "project", "local":
		if p.ProjectPath == "" || cwd == "" {
			return false
		}
		return isSameOrUnder(p.ProjectPath, cwd)
	default:
		return true
	}
}

// isSameOrUnder reports whether dir is root or a directory under it.
func isSameOrUnder(root, dir string) bool {
	root, dir = resolvePath(root), resolvePath(dir)
	rel, err := filepath.Rel(root, dir)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return true
	}
	// Two spellings can still name one directory: a different case on macOS's
	// case-insensitive APFS (Rel folds case only on Windows), a Windows 8.3
	// short name, or a bind mount. Compare file identity with dir and each of
	// its ancestors; the walk is bounded by dir's depth.
	rootInfo, err := os.Stat(root)
	if err != nil {
		return false
	}
	for {
		if info, err := os.Stat(dir); err == nil && os.SameFile(rootInfo, info) {
			return true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// resolvePath cleans path and resolves symlinks when it can (e.g. macOS /var
// -> /private/var), so a project path and a working directory spelled
// differently still compare equal.
func resolvePath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}

// DatabricksPluginVersion reports the recorded version of the databricks plugin
// in the agent's own plugin manifest and whether the plugin is installed at all,
// in any scope or project. This catches installs done through `databricks
// aitools install` and a direct `<agent> plugin install`, unlike the CLI's own
// state file, which only records CLI-driven installs. Each agent's manifest
// format differs, so parsing is delegated to a per-agent reader; agents without
// a verified reader report ("", false), i.e. "not installed as far as we can
// tell". When the plugin is recorded for more than one scope or marketplace,
// the highest recorded version is returned (they match in the common
// single-scope case, but the cache can retain a stale scope); the version is ""
// when installed but unversioned.
func (a *Agent) DatabricksPluginVersion(ctx context.Context) (string, bool) {
	if a.pluginInstalls == nil {
		return "", false
	}
	installs, ok := a.pluginInstalls(ctx, a)
	if !ok {
		return "", false
	}
	best := ""
	for _, install := range installs {
		// Compare as semver (versions are unprefixed, e.g. "0.2.9").
		if install.Version != "" && (best == "" || semver.Compare("v"+install.Version, "v"+best) > 0) {
			best = install.Version
		}
	}
	return best, true
}

// databricksPluginActiveIn reports whether the agent's own manifest has a
// databricks plugin record that applies in cwd (see pluginInstall.appliesTo).
func (a *Agent) databricksPluginActiveIn(ctx context.Context, cwd string) bool {
	if a.pluginInstalls == nil {
		return false
	}
	installs, _ := a.pluginInstalls(ctx, a)
	for _, install := range installs {
		if install.appliesTo(cwd) {
			return true
		}
	}
	return false
}

// claudePluginInstalls reads the databricks plugin's records from Claude Code's
// installed_plugins.json (<ConfigDir>/plugins/), which keys each entry as
// "<id>@<marketplace>" (e.g. "databricks@claude-plugins-official") with one
// record per install scope. The plugin can be installed from more than one
// marketplace, so the records of every databricks entry are returned. Any
// read/parse failure reports (nil, false), so an unreadable manifest reads as
// "not installed".
func claudePluginInstalls(ctx context.Context, a *Agent) ([]pluginInstall, bool) {
	configDir, err := a.ConfigDir(ctx)
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(filepath.Join(configDir, "plugins", pluginManifestFile))
	if err != nil {
		return nil, false
	}
	var manifest struct {
		Plugins map[string][]pluginInstall `json:"plugins"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, false
	}
	var all []pluginInstall
	found := false
	for key, installs := range manifest.Plugins {
		id, _, _ := strings.Cut(key, "@")
		if id == databricksPluginID {
			all = append(all, installs...)
			found = true
		}
	}
	return all, found
}
