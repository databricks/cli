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
	// DetectFile, when set, is a marker file under ConfigDir that must exist for the
	// agent to count as installed, instead of the bare config directory (used when
	// the config dir is shared with another product; see geminiDetectFile).
	DetectFile string
	// Plugin describes the databricks plugin for this agent, or nil when the
	// agent has no plugin and skills files are its native delivery.
	Plugin *PluginSpec
	// pluginVersion reads the installed databricks plugin version from the
	// agent's own plugin manifest. Each agent's manifest format differs, so it is
	// set per agent (only for formats we have verified); when nil,
	// DatabricksPluginVersion reports no version. New agents extend support by
	// providing their own reader here.
	pluginVersion func(ctx context.Context, a *Agent) (string, bool)
}

// Detected reports whether the agent is installed: its config directory exists,
// or its DetectFile marker or installed Databricks skills exist when one is set.
func (a *Agent) Detected(ctx context.Context) bool {
	dir, err := a.ConfigDir(ctx)
	if err != nil {
		return false
	}
	target := dir
	if a.DetectFile != "" {
		target = filepath.Join(dir, a.DetectFile)
	}
	if _, err = os.Stat(target); err == nil {
		return true
	}
	if a.DetectFile == "" {
		return false
	}
	skillsDir, err := a.SkillsDir(ctx)
	return err == nil && HasDatabricksSkillsIn(skillsDir)
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
	NamePi          = "pi"
	NameGemini      = "gemini"
	NameGoose       = "goose"
)

// geminiDetectFile is the project registry Gemini CLI writes at ~/.gemini/projects.json
// on real use. Detection keys on it, not the bare ~/.gemini directory, because
// Antigravity's ~/.gemini/antigravity subtree makes ~/.gemini exist without Gemini
// CLI being installed. (installation_id is not reliably present.)
const geminiDetectFile = "projects.json"

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
		pluginVersion:        claudePluginVersion,
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
	{
		Name:                 NamePi,
		DisplayName:          "Pi",
		ConfigDir:            piConfigDir,
		SupportsProjectScope: true,
		ProjectConfigDir:     ".pi",
		Binary:               "pi",
		// Pi reads agent skills (SKILL.md) but has no databricks plugin, so it is
		// skills-only (Plugin nil).
	},
	{
		Name:                 NameGemini,
		DisplayName:          "Gemini CLI",
		ConfigDir:            geminiConfigDir,
		SupportsProjectScope: true,
		ProjectConfigDir:     ".gemini",
		Binary:               "gemini",
		// Gemini CLI reads agent skills (SKILL.md) but has no databricks plugin, so
		// it is skills-only (Plugin nil).
		DetectFile: geminiDetectFile,
	},
	{
		Name:                 NameGoose,
		DisplayName:          "Goose",
		ConfigDir:            gooseConfigDir,
		SupportsProjectScope: true,
		ProjectConfigDir:     ".goose",
		Binary:               "goose",
		// Goose reads agent skills (SKILL.md) but has no databricks plugin, so it is
		// skills-only (Plugin nil).
	},
}

// piConfigDir returns Pi's agent config directory: PI_CODING_AGENT_DIR when set,
// else ~/.pi/agent. Mirroring Pi's own override keeps skills where Pi reads them
// when a launcher (e.g. ucode) relocates its home.
// See getAgentDir in @earendil-works/pi-coding-agent (config.ts).
func piConfigDir(ctx context.Context) (string, error) {
	if dir := env.Get(ctx, "PI_CODING_AGENT_DIR"); dir != "" {
		return dir, nil
	}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".pi", "agent"), nil
}

// geminiConfigDir returns Gemini CLI's config directory: <GEMINI_CLI_HOME>/.gemini
// when set, else ~/.gemini. Honoring Gemini's own override keeps skills where it
// reads them under a relocated home (e.g. ucode).
// https://github.com/google-gemini/gemini-cli/blob/main/docs/reference/configuration.md
func geminiConfigDir(ctx context.Context) (string, error) {
	root := env.Get(ctx, "GEMINI_CLI_HOME")
	if root == "" {
		home, err := env.UserHomeDir(ctx)
		if err != nil {
			return "", err
		}
		root = home
	}
	return filepath.Join(root, ".gemini"), nil
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

// gooseConfigDir returns Goose's config directory, matching how Goose resolves it
// so skills land where it reads them, including under a relocated root. The Windows
// path keeps the legacy "Block" segment for backwards compatibility.
// See crates/goose/src/config/paths.rs (etcetera crate). https://block.github.io/goose/
func gooseConfigDir(ctx context.Context) (string, error) {
	if root := env.Get(ctx, "GOOSE_PATH_ROOT"); filepath.IsAbs(root) {
		return filepath.Join(root, "config"), nil
	}
	if runtime.GOOS == "windows" {
		if appData := env.Get(ctx, "APPDATA"); appData != "" {
			return filepath.Join(appData, "Block", "goose", "config"), nil
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
	return filepath.Join(xdg, "goose"), nil
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

// DetectProjectInstalled returns project-scope agents that already have Databricks
// skills in the current project. Config-dir detection is home-based, so it misses
// project-local installs; update uses this to also refresh those.
func DetectProjectInstalled(cwd string) []*Agent {
	var installed []*Agent
	for _, a := range Registry {
		if a.SupportsProjectScope && HasDatabricksSkillsIn(a.ProjectSkillsDir(cwd)) {
			installed = append(installed, a)
		}
	}
	return installed
}

// SupportedNames returns every agent's display name in registry order, so the
// "Supported agents" messages can't drift as agents are added.
func SupportedNames() []string {
	names := make([]string, len(Registry))
	for i, a := range Registry {
		names[i] = a.DisplayName
	}
	return names
}

// SkillsOnlyNames returns the display names of skills-only agents (Plugin == nil)
// in registry order, so the install help can't drift as they are added.
func SkillsOnlyNames() []string {
	var names []string
	for _, a := range Registry {
		if a.Plugin == nil {
			names = append(names, a.DisplayName)
		}
	}
	return names
}
