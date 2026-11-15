package agents

import (
	"context"
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/folders"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// providerAliases maps the SDK agent products (useragent.AgentProvider values)
// whose names differ from the registry name. Every other product is looked up
// by its own name.
// https://github.com/databricks/databricks-sdk-go/blob/main/useragent/agent.go
var providerAliases = map[string]string{
	"copilot-cli": NameCopilot,
	"gemini-cli":  NameGemini,
}

// AgentForProvider returns the registry agent for an SDK agent product
// (useragent.AgentProvider value), or nil when the product has no registry
// entry, is "multiple", or is empty.
func AgentForProvider(provider string) *Agent {
	if name, ok := providerAliases[provider]; ok {
		provider = name
	}
	return ByName(provider)
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

// String returns the value reported in the aitools-agent User-Agent segment.
func (t Tooling) String() string {
	switch t {
	case ToolingPlugin:
		return "plugin"
	case ToolingSkills:
		return "skills"
	default:
		return "none"
	}
}

// DatabricksTooling reports the Databricks plugin or skills in this agent's own
// locations; read errors count as not installed. Claude Code's manifest is
// authoritative because `claude plugin uninstall` leaves a stale CLI install
// record. For Codex and Copilot, pluginRecorded (the CLI install state) is the
// only plugin source, so it misses plugins installed with the agent's own CLI
// and includes plugins later removed with it. The shared skills store is not consulted: it cannot attribute skills.
// The install hint's claudeToolingInstalled differs on purpose: it stays quiet
// for any trace of an install, while this reports what the agent loads.
func (a *Agent) DatabricksTooling(ctx context.Context, pluginRecorded func() bool) Tooling {
	// An unknown working directory skips the project lookups below.
	cwd := ""
	if wd, err := os.Getwd(); err == nil {
		cwd = resolvePath(wd)
	}
	switch {
	case a.pluginInstalls != nil:
		if active, _ := a.DatabricksPluginActiveIn(ctx, cwd); active {
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
// in a subdirectory: https://code.claude.com/docs/en/skills ("in every parent
// directory up to the repository root"; up to the worktree root in a linked
// worktree). Outside a repository only cwd is returned. cwd must be
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
