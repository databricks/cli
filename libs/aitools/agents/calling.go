package agents

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/folders"
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

// DatabricksTooling reports the Databricks plugin or skills in this agent's own
// locations; read errors count as not installed. Claude Code's manifest is
// authoritative because `claude plugin uninstall` leaves a stale CLI install
// record. For Codex and Copilot, pluginRecorded (the CLI install state) is the
// only plugin source, so it includes plugins later removed with the agent's own
// CLI. The shared skills store is not consulted: it cannot attribute skills.
func (a *Agent) DatabricksTooling(ctx context.Context, pluginRecorded func() bool) Tooling {
	// Getwd returns "" on error, which skips the project lookups below.
	cwd, err := os.Getwd()
	if err == nil {
		cwd = resolvePath(cwd)
	}
	switch {
	case a.pluginInstalls != nil:
		if a.databricksPluginActiveIn(ctx, cwd) {
			return ToolingPlugin
		}
	case a.Plugin != nil:
		if pluginRecorded() {
			return ToolingPlugin
		}
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
// in a subdirectory. Outside a repository only cwd is returned. cwd must be
// resolved (see resolvePath), so a working directory reached through a symlink
// walks the parents of the directory it points to.
func projectDirs(cwd string) []string {
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
