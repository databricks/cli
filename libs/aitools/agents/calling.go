package agents

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/folders"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// SDK agent products (useragent.AgentProvider values) whose names differ from
// the registry name.
// https://github.com/databricks/databricks-sdk-go/blob/main/useragent/agent.go
const (
	providerCopilot = "copilot-cli"
	providerGemini  = "gemini-cli"
)

// providerToRegistry maps SDK agent products to registry agent names. Products
// missing here have no registry entry, so the CLI cannot install into them.
var providerToRegistry = map[string]string{
	NameClaudeCode:  NameClaudeCode,
	NameCursor:      NameCursor,
	NameCodex:       NameCodex,
	NameOpenCode:    NameOpenCode,
	providerCopilot: NameCopilot,
	NameAntigravity: NameAntigravity,
	NamePi:          NamePi,
	providerGemini:  NameGemini,
	NameGoose:       NameGoose,
	NameKiro:        NameKiro,
}

// AgentForProvider returns the registry agent for an SDK agent product
// (useragent.AgentProvider value), or nil when the product has no registry
// entry, is "multiple", or is empty.
func AgentForProvider(provider string) *Agent {
	name, ok := providerToRegistry[provider]
	if !ok {
		return nil
	}
	return ByName(name)
}

// CallingAgent returns the registry agent driving the CLI, or nil when no
// agent is detected, several are, or the agent is one the CLI cannot install
// into.
func CallingAgent() *Agent {
	return AgentForProvider(useragent.AgentProvider())
}

// Tooling is what Databricks AI tooling an agent has in its own location.
type Tooling int

const (
	ToolingNone Tooling = iota
	ToolingSkills
	ToolingPlugin
)

// DatabricksTooling reports the Databricks AI tooling installed for this agent
// in the agent's own locations, so tooling installed for one agent never counts
// for another:
//   - the databricks plugin in the agent's own plugin manifest, for agents with
//     a verified manifest reader (Claude Code), counting a project or local
//     scope record only inside its project. That manifest is authoritative:
//     a plugin removed with the agent's own CLI leaves a stale CLI state record
//     behind, so the record is not consulted for these agents. A plugin that
//     is installed but disabled in the agent's settings still counts;
//   - for agents without a reader (Codex, Copilot), the plugin recorded for
//     this agent in the global CLI install state;
//   - a databricks* skill directory in the agent's global skills dir, or in its
//     project skills dir. This covers `aitools install`, `--skills-only` and
//     `apps init`, which all place skills there (as symlinks or copies).
//
// Project skills dirs are looked up in the working directory and its parents
// up to the repository root (see projectDirs), because the agent's project is
// often a parent of the directory the CLI runs in.
//
// The shared ~/.databricks/aitools/skills store is deliberately not consulted:
// it holds the skills of every agent installed into, so it cannot tell which
// agent has them. Installs the CLI did not perform and that leave no trace in
// these locations (e.g. a Codex plugin installed with `codex` directly) read as
// ToolingNone. Read errors read as "not installed".
func (a *Agent) DatabricksTooling(ctx context.Context) Tooling {
	cwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		cwd = ""
	}
	if a.pluginInstalls != nil {
		if a.databricksPluginActiveIn(ctx, cwd) {
			return ToolingPlugin
		}
	} else if pluginRecorded(ctx, a.Name) {
		return ToolingPlugin
	}
	if dir, err := a.SkillsDir(ctx); err == nil && HasDatabricksSkillsIn(dir) {
		return ToolingSkills
	}
	if a.SupportsProjectScope && cwd != "" {
		for _, dir := range projectDirs(cwd) {
			if HasDatabricksSkillsIn(a.ProjectSkillsDir(dir)) {
				return ToolingSkills
			}
		}
	}
	return ToolingNone
}

// projectDirs returns cwd and each of its parents up to the repository root,
// the nearest directory holding .git (a file in a git worktree, so the walk
// stops at the worktree root). Claude Code loads project skills the same way,
// so an agent started at the repository root still has them when the CLI runs
// in a subdirectory. Outside a repository only cwd is returned. cwd is resolved
// first (see resolvePath), so a working directory reached through a symlink
// walks the parents of the directory it points to.
func projectDirs(cwd string) []string {
	cwd = resolvePath(cwd)
	dirs := []string{cwd}
	root, err := folders.FindDirWithLeaf(cwd, ".git")
	if err != nil {
		return dirs
	}
	for dir := cwd; dir != root; {
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
		dirs = append(dirs, dir)
	}
	return dirs
}

// HasDatabricksTooling reports whether the agent has the databricks plugin or
// Databricks skills in its own locations (see DatabricksTooling).
func (a *Agent) HasDatabricksTooling(ctx context.Context) bool {
	return a.DatabricksTooling(ctx) != ToolingNone
}

// installStateFile is the CLI install state written by libs/aitools/installer
// (InstallState) in each scope's canonical skills dir. Only its plugins map is
// read here; the installer package imports this one, so it cannot be reused.
const installStateFile = ".state.json"

// pluginRecorded reports whether the global CLI install state records a
// databricks plugin install for the agent. Only the global state is read: the
// CLI installs plugins in project scope only for agents with SupportsProjectScope
// (see mapAgentScope in cmd/aitools), and those have a manifest reader.
func pluginRecorded(ctx context.Context, agentName string) bool {
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return false
	}
	dir := filepath.Join(home, CanonicalSkillsDir)
	data, err := os.ReadFile(filepath.Join(dir, installStateFile))
	if err != nil {
		return false
	}
	var state struct {
		Plugins map[string]json.RawMessage `json:"plugins"`
	}
	if err := json.Unmarshal(data, &state); err != nil {
		log.Debugf(ctx, "Could not parse AI tools install state in %s: %v", dir, err)
		return false
	}
	_, ok := state.Plugins[agentName]
	return ok
}
