package agents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/semver"
)

// pluginManifestFile is the file an agent's own plugin CLI writes to record
// installed plugins. Claude Code stores it at <configDir>/plugins and keys each
// entry as "<id>@<marketplace>" (e.g. "databricks@claude-plugins-official").
const pluginManifestFile = "installed_plugins.json"

// Claude Code's install scopes that apply only inside one project.
const (
	claudeScopeProject = "project"
	claudeScopeLocal   = "local"
)

// pluginInstall is one install record in an agent's plugin manifest. A plugin
// installed in more than one scope has several records. Claude Code documents
// the record's scope and version and what each scope covers:
// https://code.claude.com/docs/en/plugins/loading#check-which-stage-a-plugin-reached
// https://code.claude.com/docs/en/plugins/install#choose-an-install-scope
type pluginInstall struct {
	Version string `json:"version"`
	// Scope is Claude Code's install scope: "managed", "user", "project" or
	// "local".
	Scope string `json:"scope"`
	// ProjectPath is the project a project or local scope record applies to.
	// It is not in Claude Code's public docs; Claude Code writes it for those
	// scopes.
	ProjectPath string `json:"projectPath"`
	// Marketplace is the marketplace of the manifest entry the record is
	// under, from its "<id>@<marketplace>" key.
	Marketplace string `json:"-"`
}

// PluginInstall is one databricks plugin install in an agent's own manifest:
// the marketplace it came from and the agent's install scope, which together
// name the install to the agent's own plugin CLI.
type PluginInstall struct {
	Marketplace string
	Scope       string
}

// appliesTo reports whether the agent loads this install when run in cwd:
// always for machine-wide scopes, and for project and local scopes only inside
// the recorded project (cwd is the project or a directory under it). A record
// with an unknown scope counts, so a new machine-wide scope is not missed. cwd
// must be resolved (see resolvePath).
func (p pluginInstall) appliesTo(cwd string) bool {
	switch p.Scope {
	case claudeScopeProject, claudeScopeLocal:
		if p.ProjectPath == "" || cwd == "" {
			return false
		}
		return isSameOrUnder(p.ProjectPath, cwd)
	default:
		return true
	}
}

// isSameOrUnder reports whether dir is root or a directory under it. dir must
// be resolved (see resolvePath); root is resolved here.
func isSameOrUnder(root, dir string) bool {
	root = resolvePath(root)
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
	installs, found, err := a.pluginInstalls(ctx, a)
	if err != nil || !found {
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

// HasPluginManifestReader reports whether the CLI can read the agent's own
// plugin manifest (Claude Code), not whether the manifest file exists. For such
// agents the manifest is authoritative: a plugin removed with the agent's own
// CLI leaves the CLI's state record behind.
func (a *Agent) HasPluginManifestReader() bool {
	return a.pluginInstalls != nil
}

// DatabricksPluginActiveIn reports whether the agent's own manifest has a
// databricks plugin record that applies in cwd (see pluginInstall.appliesTo).
// A plugin disabled in the agent's settings still counts.
// With cwd "", only machine-wide records (not project or local ones) count.
// cwd is resolved here, so a working directory reached through a symlink is
// matched by the directory it points to. A missing manifest reports false; an
// error means the manifest exists but could not be read, so the answer is
// unknown.
func (a *Agent) DatabricksPluginActiveIn(ctx context.Context, cwd string) (bool, error) {
	installs, err := a.DatabricksPluginInstallsIn(ctx, cwd)
	return len(installs) > 0, err
}

// DatabricksPluginInstallsIn returns the databricks plugin records that
// DatabricksPluginActiveIn counts for cwd.
func (a *Agent) DatabricksPluginInstallsIn(ctx context.Context, cwd string) ([]PluginInstall, error) {
	if cwd != "" {
		cwd = resolvePath(cwd)
	}
	return a.databricksPluginInstalls(ctx, func(install pluginInstall) bool {
		return install.appliesTo(cwd)
	})
}

// DatabricksProjectPluginInstallsIn returns the project-scope databricks plugin
// records for the project that contains cwd: the plugin a project shares
// through its repository. Machine-wide and local records are personal and do
// not count. A missing manifest has none; an error means the manifest exists
// but could not be read.
func (a *Agent) DatabricksProjectPluginInstallsIn(ctx context.Context, cwd string) ([]PluginInstall, error) {
	cwd = resolvePath(cwd)
	return a.databricksPluginInstalls(ctx, func(install pluginInstall) bool {
		return install.Scope == claudeScopeProject && install.appliesTo(cwd)
	})
}

// databricksPluginInstalls returns the databricks plugin records in the
// agent's own manifest that match keep. An agent without a manifest reader or
// a missing manifest has none.
func (a *Agent) databricksPluginInstalls(ctx context.Context, keep func(pluginInstall) bool) ([]PluginInstall, error) {
	if a.pluginInstalls == nil {
		return nil, nil
	}
	installs, _, err := a.pluginInstalls(ctx, a)
	if err != nil {
		return nil, err
	}
	var out []PluginInstall
	for _, install := range installs {
		if keep(install) {
			out = append(out, PluginInstall{Marketplace: install.Marketplace, Scope: install.Scope})
		}
	}
	return out, nil
}

// claudePluginInstalls reads the databricks plugin's records from Claude Code's
// installed_plugins.json (<ConfigDir>/plugins/), which keys each entry as
// "<id>@<marketplace>" (e.g. "databricks@claude-plugins-official") with one
// record per install scope. The plugin can be installed from more than one
// marketplace, so the records of every databricks entry are returned, and found
// reports whether there is any such entry. A missing manifest is not found; a
// manifest that cannot be read or parsed returns an error.
func claudePluginInstalls(ctx context.Context, a *Agent) ([]pluginInstall, bool, error) {
	configDir, err := a.ConfigDir(ctx)
	if err != nil {
		return nil, false, err
	}
	data, err := os.ReadFile(filepath.Join(configDir, "plugins", pluginManifestFile))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var manifest struct {
		Plugins map[string][]pluginInstall `json:"plugins"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, false, fmt.Errorf("parse %s: %w", pluginManifestFile, err)
	}
	var all []pluginInstall
	found := false
	// Sorted keys, so the records of several marketplaces come in a stable
	// order.
	for _, key := range slices.Sorted(maps.Keys(manifest.Plugins)) {
		id, marketplace, _ := strings.Cut(key, "@")
		if id != databricksPluginID {
			continue
		}
		for _, install := range manifest.Plugins[key] {
			install.Marketplace = marketplace
			all = append(all, install)
		}
		found = true
	}
	return all, found, nil
}
