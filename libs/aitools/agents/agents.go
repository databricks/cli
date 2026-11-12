package agents

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
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
	// (e.g. "databricks/databricks-agent-skills").
	Source string
	// Shared marks a marketplace we do not own. It is still added when missing — an
	// unregistered marketplace can't be refreshed or installed from — but is never
	// de-registered on uninstall, since other plugins may rely on it.
	Shared bool
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
	// MandatoryFile is a marker file required to confirm installation. It is a
	// relative filename in ConfigDir. Use it when the presence of ConfigDir is not
	// enough to determine if the agent is installed, such as when ConfigDir is
	// shared with other products.
	MandatoryFile string
	// Plugin describes the databricks plugin for this agent, or nil when the
	// agent has no plugin and skills files are its native delivery.
	Plugin *PluginSpec
	// MCP describes where the agent keeps user-scope MCP server entries, or nil
	// when this CLI does not write them (Pi, Goose, Kiro).
	MCP *mcpconfig.Spec
	// pluginInstalls reads the databricks plugin's install records from the
	// agent's own plugin manifest. Each agent's manifest format differs, so it is
	// set per agent (only for formats we have verified); when nil,
	// DatabricksPluginVersion reports no version and DatabricksTooling falls
	// back to the CLI install state. New agents extend support by providing
	// their own reader here. found is false when the manifest has no databricks
	// entry or does not exist; an error means it exists but could not be read.
	pluginInstalls func(ctx context.Context, a *Agent) (installs []pluginInstall, found bool, err error)
}

// Detected reports whether the agent is installed: its config directory exists,
// or its mandatory file or installed Databricks skills exist when one is set.
func (a *Agent) Detected(ctx context.Context) bool {
	dir, err := a.ConfigDir(ctx)
	if err != nil {
		return false
	}
	target := dir
	if a.MandatoryFile != "" {
		target = filepath.Join(dir, a.MandatoryFile)
	}
	if _, err = os.Stat(target); err == nil {
		return true
	}
	if a.MandatoryFile == "" {
		return false // ConfigDir does not exist
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
	NameKiro        = "kiro"
)

// MCP entry keys shared by the agents' MCP config specs.
const (
	// mcpServersKey is the key of the server table in the JSON MCP config files
	// that use it (every agent but Codex and OpenCode).
	mcpServersKey = "mcpServers"
	// localServerType is the "type" of a local stdio server in Copilot and
	// OpenCode. Claude Code writes "stdio" instead.
	localServerType = "local"
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

	claudeOfficialMarketplace    = "claude-plugins-official"
	claudeOfficialMarketplaceSrc = "anthropics/claude-plugins-official"
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

func claudePlugin() *PluginSpec {
	return &PluginSpec{
		Marketplace: claudeOfficialMarketplace,
		ID:          databricksPluginID,
		Source:      claudeOfficialMarketplaceSrc,
		Shared:      true,
	}
}

// Registry contains all supported agents. It holds pointers so callers can take
// an *Agent without the &Registry[i] dance and call its pointer-receiver methods.
var Registry = []*Agent{
	{
		Name:                 NameClaudeCode,
		DisplayName:          "Claude Code",
		ConfigDir:            claudeConfigDir,
		SupportsProjectScope: true,
		ProjectConfigDir:     ".claude",
		Binary:               "claude",
		Plugin:               claudePlugin(),
		pluginInstalls:       claudePluginInstalls,
		MCP: &mcpconfig.Spec{
			Path:   claudeMCPPath,
			Format: mcpconfig.FormatJSON,
			Root:   mcpServersKey,
			Entry:  claudeMCPEntry,
		},
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
		MCP: &mcpconfig.Spec{
			Path:   homeSubdir(".cursor", "mcp.json"),
			Format: mcpconfig.FormatJSON,
			Root:   mcpServersKey,
			Entry:  stdioEntry,
		},
	},
	{
		Name:        NameCodex,
		DisplayName: "Codex CLI",
		ConfigDir:   homeSubdir(".codex"),
		Binary:      "codex",
		Plugin:      databricksPlugin(),
		MCP: &mcpconfig.Spec{
			Path:   codexMCPPath,
			Format: mcpconfig.FormatTOML,
			Root:   "mcp_servers",
			Entry:  stdioEntry,
		},
	},
	{
		Name:        NameOpenCode,
		DisplayName: "OpenCode",
		ConfigDir:   openCodeConfigDir,
		Binary:      "opencode",
		// OpenCode exposes an `opencode plugin <module>` command, but that's an
		// npm-module system, not the agent-skills marketplace, so OpenCode stays
		// skills-only (Plugin nil).
		MCP: &mcpconfig.Spec{
			Path:   openCodeMCPPath,
			Format: mcpconfig.FormatJSON,
			Root:   "mcp.servers",
			Entry:  openCodeMCPEntry,
		},
	},
	{
		Name:        NameCopilot,
		DisplayName: "GitHub Copilot",
		ConfigDir:   homeSubdir(".copilot"),
		Binary:      "copilot",
		Plugin:      databricksPlugin(),
		MCP: &mcpconfig.Spec{
			Path:   copilotMCPPath,
			Format: mcpconfig.FormatJSON,
			Root:   mcpServersKey,
			Entry:  copilotMCPEntry,
		},
	},
	{
		Name:         NameAntigravity,
		DisplayName:  "Antigravity",
		ConfigDir:    homeSubdir(".gemini", "antigravity"),
		SkillsSubdir: "global_skills",
		// Antigravity is IDE-only with no CLI binary, so it has no plugin path.
		MCP: &mcpconfig.Spec{
			Path:   antigravityMCPPath,
			Format: mcpconfig.FormatJSON,
			Root:   mcpServersKey,
			Entry:  stdioEntry,
		},
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
		// Gemini writes projects.json after real use. Antigravity shares ~/.gemini,
		// and installation_id is not reliable, so detection uses this Gemini-only file.
		MandatoryFile: "projects.json",
		MCP: &mcpconfig.Spec{
			Path:   geminiMCPPath,
			Format: mcpconfig.FormatJSON,
			Root:   mcpServersKey,
			Entry:  stdioEntry,
		},
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
	{
		Name:                 NameKiro,
		DisplayName:          "Kiro",
		ConfigDir:            homeSubdir(".kiro"),
		SupportsProjectScope: true,
		ProjectConfigDir:     ".kiro",
		Binary:               "kiro",
		// Kiro reads agent skills (SKILL.md) but has no databricks plugin, so it is
		// skills-only (Plugin nil). Kiro's own picker text documents both scopes:
		// "Skills can be added to ~/.kiro/skills/ (user-level) or .kiro/skills/
		// (workspace-level)". Kiro is IDE-first, so `kiro` is frequently absent from
		// PATH; detection then falls back to ConfigDir and reports files-only.
	},
}

// claudeConfigDir follows Claude Code's configuration directory override.
// See https://code.claude.com/docs/en/env-vars for CLAUDE_CONFIG_DIR.
func claudeConfigDir(ctx context.Context) (string, error) {
	if dir := env.Get(ctx, "CLAUDE_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	return homeSubdir(".claude")(ctx)
}

// piConfigDir returns Pi's agent config directory: PI_CODING_AGENT_DIR when set,
// else ~/.pi/agent. Mirroring Pi's own override keeps skills where Pi reads them
// when a launcher (e.g. ucode) relocates its home.
// See getAgentDir in @earendil-works/pi-coding-agent (config.ts).
func piConfigDir(ctx context.Context) (string, error) {
	if dir := env.Get(ctx, "PI_CODING_AGENT_DIR"); dir != "" {
		if dir == "~" || strings.HasPrefix(dir, "~/") || (runtime.GOOS == "windows" && strings.HasPrefix(dir, `~\`)) {
			home, err := env.UserHomeDir(ctx)
			if err != nil {
				return "", err
			}
			if dir == "~" {
				return home, nil
			}
			return filepath.Join(home, dir[2:]), nil
		}
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
		home, err := env.UserHomeDir(ctx)
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "AppData", "Roaming", "Block", "goose", "config"), nil
	}
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	xdg := env.Get(ctx, "XDG_CONFIG_HOME")
	if !filepath.IsAbs(xdg) {
		xdg = filepath.Join(home, ".config")
	}
	return filepath.Join(xdg, "goose"), nil
}

// claudeMCPPath returns Claude Code's global config file, which holds user-scope
// mcpServers next to auth state. It sits in the home directory, not in ~/.claude;
// CLAUDE_CONFIG_DIR moves it to <dir>/.claude.json.
// https://code.claude.com/docs/en/env-vars
func claudeMCPPath(ctx context.Context) (string, error) {
	if dir := env.Get(ctx, "CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, ".claude.json"), nil
	}
	return homeSubdir(".claude.json")(ctx)
}

// claudeMCPEntry matches `claude mcp add`, which always writes an env map, even an empty one.
func claudeMCPEntry(cmd string, args []string, vars map[string]string) map[string]any {
	if len(vars) == 0 {
		vars = map[string]string{}
	}
	return map[string]any{"type": "stdio", "command": cmd, "args": args, "env": vars}
}

// codexMCPPath returns Codex's config.toml, which honors CODEX_HOME.
// https://learn.chatgpt.com/docs/config-file/config-advanced
func codexMCPPath(ctx context.Context) (string, error) {
	if home := env.Get(ctx, "CODEX_HOME"); home != "" {
		return filepath.Join(home, "config.toml"), nil
	}
	return homeSubdir(".codex", "config.toml")(ctx)
}

// openCodeMCPPath returns OpenCode's config file. OpenCode reads opencode.json or
// opencode.jsonc, and the docs do not say which wins when both exist, so an
// existing opencode.jsonc is used rather than creating a second file beside it.
// The file may contain comments; a file we cannot parse is refused, never rewritten.
// https://opencode.ai/docs/config/
func openCodeMCPPath(ctx context.Context) (string, error) {
	dir, err := openCodeConfigDir(ctx)
	if err != nil {
		return "", err
	}
	jsonc := filepath.Join(dir, "opencode.jsonc")
	if _, err := os.Stat(jsonc); err == nil {
		return jsonc, nil
	}
	return filepath.Join(dir, "opencode.json"), nil
}

// openCodeMCPEntry follows the current OpenCode schema: servers live under
// mcp.servers, the env map is "environment", and there is no "enabled" key
// (the schema forbids extra keys, and an absent key means enabled).
// https://opencode.ai/docs/mcp-servers/
func openCodeMCPEntry(cmd string, args []string, vars map[string]string) map[string]any {
	entry := map[string]any{"type": localServerType, "command": append([]string{cmd}, args...)}
	if len(vars) > 0 {
		entry["environment"] = vars
	}
	return entry
}

// copilotMCPPath returns Copilot CLI's user config. COPILOT_HOME replaces ~/.copilot.
// https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-config-dir-reference
func copilotMCPPath(ctx context.Context) (string, error) {
	if home := env.Get(ctx, "COPILOT_HOME"); home != "" {
		return filepath.Join(home, "mcp-config.json"), nil
	}
	return homeSubdir(".copilot", "mcp-config.json")(ctx)
}

// copilotMCPEntry is the stdio entry Copilot CLI documents, with all tools enabled.
// https://docs.github.com/en/copilot/how-tos/copilot-cli/customize-copilot/add-mcp-servers
func copilotMCPEntry(cmd string, args []string, vars map[string]string) map[string]any {
	entry := stdioEntry(cmd, args, vars)
	entry["type"] = localServerType
	entry["tools"] = []string{"*"}
	return entry
}

// geminiMCPPath returns Gemini CLI's user settings, which hold mcpServers. The
// settings directory follows geminiConfigDir, which honors GEMINI_CLI_HOME.
// https://geminicli.com/docs/tools/mcp-server/
func geminiMCPPath(ctx context.Context) (string, error) {
	dir, err := geminiConfigDir(ctx)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// antigravityMCPPath is the global MCP file in Antigravity's docs, for both the
// CLI and the IDE. It is not the skills directory (~/.gemini/antigravity).
// https://antigravity.google/docs/mcp
func antigravityMCPPath(ctx context.Context) (string, error) {
	return homeSubdir(".gemini", "config", "mcp_config.json")(ctx)
}

// stdioEntry is the command/args entry used by Cursor, Codex, Gemini and
// Antigravity. env is omitted when empty. Antigravity's "disabled" key is
// optional and defaults to false, so it is not written.
func stdioEntry(cmd string, args []string, vars map[string]string) map[string]any {
	entry := map[string]any{"command": cmd, "args": args}
	if len(vars) > 0 {
		entry["env"] = vars
	}
	return entry
}

// SupportsMCP reports whether this CLI writes the agent's MCP server entries.
func (a *Agent) SupportsMCP() bool {
	return a.MCP != nil
}

// MCPAgents returns the agents whose MCP server entries this CLI writes, in
// registry order.
func MCPAgents() []*Agent {
	var agents []*Agent
	for _, a := range Registry {
		if a.SupportsMCP() {
			agents = append(agents, a)
		}
	}
	return agents
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
