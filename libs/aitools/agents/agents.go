package agents

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	"github.com/databricks/cli/libs/env"
)

// PluginSpec describes the databricks plugin for an agent. A non-nil
// Agent.Plugin means the agent has a databricks plugin the CLI can install
// headlessly (Claude Code, Codex, Copilot); a nil Plugin means raw skill files
// are the only delivery (OpenCode, Antigravity, Cursor).
type PluginSpec struct {
	// Marketplace is the marketplace registry name the plugin is served from,
	// as registered by `<agent> plugin marketplace add` (e.g. "databricks-agent-skills").
	Marketplace string
	// ID is the plugin identifier that is installed/enabled (e.g. "databricks").
	ID string
	// Source is the argument passed to `<agent> plugin marketplace add`
	// (e.g. "databricks/databricks-agent-skills"). Empty marks a built-in
	// marketplace that must not be added or de-registered.
	Source string
}

// Agent defines a supported coding agent.
type Agent struct {
	Name        string
	DisplayName string
	// ConfigDir returns the agent's config directory (e.g., ~/.claude).
	// Used for detection and as base for skills directory.
	ConfigDir func(ctx context.Context) (string, error)
	// SkillsSubdir is the subdirectory within ConfigDir for skills (default: "skills").
	SkillsSubdir string
	// SupportsProjectScope indicates whether this agent supports project-scoped skills.
	// When true, skills can be installed relative to the project root.
	SupportsProjectScope bool
	// ProjectConfigDir is the config directory name relative to a project root
	// (e.g., ".claude"). Only used when SupportsProjectScope is true.
	ProjectConfigDir string
	// Binary is the agent's CLI binary name on PATH, used by exec.LookPath for
	// plugin-capability detection and as the program for the plugin probe.
	// Empty for agents with no CLI binary (Antigravity is IDE-only).
	Binary string
	// Plugin describes the databricks plugin for this agent, or nil when the
	// agent has no plugin and skills files are its native delivery.
	Plugin *PluginSpec
}

// Detected returns true if the agent is installed on the system.
func (a *Agent) Detected(ctx context.Context) bool {
	dir, err := a.ConfigDir(ctx)
	if err != nil {
		return false
	}
	_, err = os.Stat(dir)
	return err == nil
}

// SkillsDir returns the full path to the agent's skills directory.
func (a *Agent) SkillsDir(ctx context.Context) (string, error) {
	configDir, err := a.ConfigDir(ctx)
	if err != nil {
		return "", err
	}
	subdir := a.SkillsSubdir
	if subdir == "" {
		subdir = "skills"
	}
	return filepath.Join(configDir, subdir), nil
}

// homeSubdir returns a function that computes ~/subpath.
func homeSubdir(subpath ...string) func(ctx context.Context) (string, error) {
	return func(ctx context.Context) (string, error) {
		home, err := env.UserHomeDir(ctx)
		if err != nil {
			return "", err
		}
		parts := append([]string{home}, subpath...)
		return filepath.Join(parts...), nil
	}
}

// ProjectSkillsDir returns the project-scoped skills directory for this agent.
// Only valid for agents where SupportsProjectScope is true.
func (a *Agent) ProjectSkillsDir(cwd string) string {
	subdir := a.SkillsSubdir
	if subdir == "" {
		subdir = "skills"
	}
	return filepath.Join(cwd, a.ProjectConfigDir, subdir)
}

// Registry agent names. Behavior keyed on a specific agent (e.g. per-agent
// plugin command shapes) references these instead of bare string literals.
const (
	NameClaudeCode  = "claude-code"
	NameCursor      = "cursor"
	NameCodex       = "codex"
	NameOpenCode    = "opencode"
	NameCopilot     = "copilot"
	NameAntigravity = "antigravity"
)

// Databricks plugin identity, shared across the agents that ship a plugin.
// The verified install commands are e.g.
//
//	claude plugin marketplace add databricks/databricks-agent-skills
//	claude plugin install        databricks@databricks-agent-skills
const (
	databricksMarketplace = "databricks-agent-skills"
	databricksPluginID    = "databricks"
	databricksPluginSrc   = "databricks/databricks-agent-skills"

	// claudeOfficialMarketplace is Claude Code's built-in marketplace
	// (anthropics/claude-plugins-official), registered by default. The databricks
	// plugin is published there, so Claude installs from it and we never register
	// our own marketplace for Claude. An empty PluginSpec.Source marks a built-in
	// marketplace that must not be added.
	claudeOfficialMarketplace = "claude-plugins-official"
)

// databricksPlugin returns the shared plugin descriptor for an agent that
// installs from our own marketplace (Codex, Copilot).
func databricksPlugin() *PluginSpec {
	return &PluginSpec{
		Marketplace: databricksMarketplace,
		ID:          databricksPluginID,
		Source:      databricksPluginSrc,
	}
}

// claudePlugin returns Claude's plugin descriptor. Claude installs the databricks
// plugin from its built-in claude-plugins-official marketplace (Source empty), so
// the CLI doesn't register a separate databricks-agent-skills marketplace for it.
func claudePlugin() *PluginSpec {
	return &PluginSpec{
		Marketplace: claudeOfficialMarketplace,
		ID:          databricksPluginID,
		Source:      "",
	}
}

// Registry contains all supported agents. It holds pointers so callers can take
// an *Agent without the &Registry[i] dance and call its pointer-receiver methods.
var Registry = []*Agent{
	{
		Name:                 NameClaudeCode,
		DisplayName:          "Claude Code",
		ConfigDir:            homeSubdir(".claude"),
		SupportsProjectScope: true,
		ProjectConfigDir:     ".claude",
		Binary:               "claude",
		Plugin:               claudePlugin(),
	},
	{
		Name:        NameCursor,
		DisplayName: "Cursor",
		ConfigDir:   homeSubdir(".cursor"),
		// Cursor's CLI binary is `cursor-agent`, not `cursor` (the latter is an
		// IDE shim that isn't on PATH unless the user ran "install shell command").
		Binary: "cursor-agent",
		// Cursor has a databricks plugin, but it can't be installed headlessly, so
		// the CLI treats Cursor as a skills-only agent (Plugin nil) rather than
		// referencing a plugin it can't act on.
	},
	{
		Name:        NameCodex,
		DisplayName: "Codex CLI",
		ConfigDir:   homeSubdir(".codex"),
		Binary:      "codex",
		Plugin:      databricksPlugin(),
	},
	{
		Name:        NameOpenCode,
		DisplayName: "OpenCode",
		ConfigDir:   openCodeConfigDir,
		Binary:      "opencode",
		// OpenCode exposes an `opencode plugin <module>` command, but that's an
		// npm-module system, not the agent-skills marketplace, so OpenCode stays
		// skills-only (Plugin nil).
	},
	{
		Name:        NameCopilot,
		DisplayName: "GitHub Copilot",
		ConfigDir:   homeSubdir(".copilot"),
		Binary:      "copilot",
		Plugin:      databricksPlugin(),
	},
	{
		Name:         NameAntigravity,
		DisplayName:  "Antigravity",
		ConfigDir:    homeSubdir(".gemini", "antigravity"),
		SkillsSubdir: "global_skills",
		// Antigravity is IDE-only with no CLI binary, so it has no plugin path.
	},
}

// openCodeConfigDir returns OpenCode's config directory. OpenCode stores its
// config under %APPDATA%\opencode (Roaming AppData) on Windows, and honors
// XDG_CONFIG_HOME on other platforms, defaulting to ~/.config/opencode. The
// previous hardcoded ~/.config/opencode made OpenCode undetectable on Windows
// and ignored XDG_CONFIG_HOME on Linux.
// See https://opencode.ai/docs/config/ and the XDG Base Directory spec.
func openCodeConfigDir(ctx context.Context) (string, error) {
	if runtime.GOOS == "windows" {
		if appData := env.Get(ctx, "APPDATA"); appData != "" {
			return filepath.Join(appData, "opencode"), nil
		}
	}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	xdg := env.Get(ctx, "XDG_CONFIG_HOME")
	if xdg == "" {
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "opencode"), nil
}

// ByName returns the registry agent with the given name, or nil if not found.
func ByName(name string) *Agent {
	for _, a := range Registry {
		if a.Name == name {
			return a
		}
	}
	return nil
}

// DetectInstalled returns all agents that are installed on the system.
func DetectInstalled(ctx context.Context) []*Agent {
	var installed []*Agent
	for _, a := range Registry {
		if a.Detected(ctx) {
			installed = append(installed, a)
		}
	}
	return installed
}
