package installer

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os/exec"
	"strings"
	"time"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
)

// lookPath resolves a binary on PATH. It is a package-level var so tests can
// inject a fake resolver without depending on the host PATH.
var lookPath = exec.LookPath

const (
	// pluginProbeTimeout bounds the `<agent> plugin --help` capability check.
	pluginProbeTimeout = 5 * time.Second
	// pluginCmdTimeout bounds an install/update/uninstall command, which may
	// clone the marketplace repo, so it gets more headroom than the probe.
	pluginCmdTimeout = 60 * time.Second
)

// BlockedError reports that a plugin operation could not be performed for an
// agent. The command layer maps Reason to a user-facing message and decides
// whether to skip-with-warning or hard-fail (per the non-TTY policy). It never
// causes a silent fall back to skills.
type BlockedError struct {
	Agent  string
	Reason string
	Detail string
}

// Reasons a plugin operation can be blocked.
const (
	// ReasonCLINotOnPath: the agent's CLI binary is not on PATH, or its CLI does
	// not expose a working `plugin` subcommand.
	ReasonCLINotOnPath = "cli-not-on-path"
	// ReasonInstallFailed: the agent's plugin CLI ran but returned an error.
	ReasonInstallFailed = "install-failed"
	// ReasonNoPlugin: the agent has no installable plugin. Callers filter these
	// out; it is guarded here to avoid a nil dereference.
	ReasonNoPlugin = "no-plugin"
)

func (e *BlockedError) Error() string {
	if e.Detail != "" {
		return fmt.Sprintf("%s: %s: %s", e.Agent, e.Reason, e.Detail)
	}
	return fmt.Sprintf("%s: %s", e.Agent, e.Reason)
}

// runAgentCmd runs an agent CLI command with a timeout, returning stdout and any
// error. Errors are *process.ProcessError, which carries the captured stderr.
func runAgentCmd(ctx context.Context, timeout time.Duration, argv []string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return process.Background(cctx, argv)
}

// stderrOf returns the captured stderr of a failed agent command, falling back
// to the error's own message. Callers must not branch on this string.
func stderrOf(err error) string {
	if perr, ok := errors.AsType[*process.ProcessError](err); ok {
		if s := strings.TrimSpace(perr.Stderr); s != "" {
			return s
		}
	}
	return err.Error()
}

// resolveAgentBinary resolves the agent's CLI binary to an absolute path.
// It refuses a binary that resolves only relative to the current directory
// (exec.ErrDot), so a malicious ./claude is never executed.
func resolveAgentBinary(agent *agents.Agent) (string, error) {
	if agent.Binary == "" {
		return "", fmt.Errorf("%s has no CLI binary", agent.DisplayName)
	}
	path, err := lookPath(agent.Binary)
	if err != nil {
		return "", fmt.Errorf("could not resolve %s on PATH: %w", agent.Binary, err)
	}
	return path, nil
}

// installTarget is the `<plugin>@<marketplace>` argument the agent CLIs accept,
// e.g. "databricks@databricks-agent-skills".
func installTarget(spec *agents.PluginSpec) string {
	return spec.ID + "@" + spec.Marketplace
}

func recordedInstallTarget(agent *agents.Agent, rec PluginRecord) string {
	plugin := rec.Plugin
	if plugin == "" {
		plugin = agent.Plugin.ID
	}
	marketplace := rec.Marketplace
	if marketplace == "" {
		marketplace = agent.Plugin.Marketplace
	}
	return plugin + "@" + marketplace
}

// marketplaceAddArgs builds the `plugin marketplace add <source>` argv (sans
// binary). Routine registration passes spec.Source; recovery passes BuiltinAddSource.
func marketplaceAddArgs(source string) []string {
	return []string{"plugin", "marketplace", "add", source}
}

// marketplaceRegistered reports whether the named marketplace is already listed
// by the agent's plugin CLI. On any uncertainty (command unsupported, error) it
// returns true, so we never claim to have added a marketplace we didn't, which
// keeps uninstall from de-registering a marketplace another plugin may share.
// The `plugin marketplace list` output shape is pending per-agent verification;
// until then the conservative default applies.
func marketplaceRegistered(ctx context.Context, bin, marketplace string) bool {
	out, err := runAgentCmd(ctx, pluginProbeTimeout, []string{bin, "plugin", "marketplace", "list"})
	if err != nil {
		return true
	}
	return strings.Contains(out, marketplace)
}

// marketplaceRemoveArgs builds the `plugin marketplace remove <name>` argv (sans binary).
func marketplaceRemoveArgs(spec *agents.PluginSpec) []string {
	return []string{"plugin", "marketplace", "remove", spec.Marketplace}
}

func marketplaceRemoveArgsForRecord(agent *agents.Agent, rec PluginRecord) []string {
	marketplace := rec.Marketplace
	if marketplace == "" {
		marketplace = agent.Plugin.Marketplace
	}
	return []string{"plugin", "marketplace", "remove", marketplace}
}

// marketplaceUpdateArgs builds the marketplace refresh command for agents whose
// plugin CLI supports it. Claude's official marketplace can be stale locally,
// causing install/update to report that a published plugin does not exist.
func marketplaceUpdateArgs(agent *agents.Agent) []string {
	if agent.Name == agents.NameClaudeCode {
		return []string{"plugin", "marketplace", "update"}
	}
	return nil
}

// pluginInstallArgs builds the per-agent install argv (sans binary). Codex uses
// `plugin add`; Claude is the only agent that accepts `--scope`.
func pluginInstallArgs(agent *agents.Agent, scope string) []string {
	target := installTarget(agent.Plugin)
	switch agent.Name {
	case agents.NameCodex:
		return []string{"plugin", "add", target}
	case agents.NameClaudeCode:
		args := []string{"plugin", "install", target}
		if scope != "" {
			args = append(args, "--scope", scope)
		}
		return args
	default:
		return []string{"plugin", "install", target}
	}
}

// pluginUpdateSteps builds the ordered per-agent update argv sets (sans binary).
// Codex updates in two steps: refresh the marketplace, then re-add.
func pluginUpdateSteps(agent *agents.Agent, rec PluginRecord) [][]string {
	target := recordedInstallTarget(agent, rec)
	switch agent.Name {
	case agents.NameCodex:
		return [][]string{
			{"plugin", "marketplace", "upgrade"},
			{"plugin", "add", target},
		}
	case agents.NameClaudeCode:
		return [][]string{
			marketplaceUpdateArgs(agent),
			appendScopeArg([]string{"plugin", "update", target}, rec.Scope),
		}
	default:
		return [][]string{{"plugin", "update", target}}
	}
}

// pluginUninstallArgs builds the per-agent uninstall argv (sans binary).
// Codex removes with `plugin remove`; the others use `plugin uninstall`.
func pluginUninstallArgs(agent *agents.Agent, rec PluginRecord) []string {
	target := recordedInstallTarget(agent, rec)
	switch agent.Name {
	case agents.NameCodex:
		return []string{"plugin", "remove", target}
	case agents.NameClaudeCode:
		return appendScopeArg([]string{"plugin", "uninstall", target}, rec.Scope)
	default:
		return []string{"plugin", "uninstall", target}
	}
}

func pluginDisableArgs(agent *agents.Agent, rec PluginRecord) []string {
	if agent.Name != agents.NameClaudeCode {
		return nil
	}
	return appendScopeArg([]string{"plugin", "disable", recordedInstallTarget(agent, rec)}, rec.Scope)
}

func appendScopeArg(args []string, scope string) []string {
	if scope == "" {
		return args
	}
	return append(args, "--scope", scope)
}

// probePluginCLI resolves the agent's binary and confirms its CLI exposes the
// plugin subcommand, so we don't register a marketplace on a CLI that can't
// install plugins. Returns the resolved absolute path.
func probePluginCLI(ctx context.Context, agent *agents.Agent) (string, error) {
	bin, err := resolveAgentBinary(agent)
	if err != nil {
		return "", &BlockedError{Agent: agent.Name, Reason: ReasonCLINotOnPath, Detail: err.Error()}
	}
	if _, err := runAgentCmd(ctx, pluginProbeTimeout, []string{bin, "plugin", "--help"}); err != nil {
		return "", &BlockedError{Agent: agent.Name, Reason: ReasonCLINotOnPath, Detail: stderrOf(err)}
	}
	return bin, nil
}

// InstallPluginForAgent registers the databricks marketplace and installs the
// plugin through the agent's own CLI, returning the record to persist in state.
// It never falls back to skills: a blocked install returns a *BlockedError.
func InstallPluginForAgent(ctx context.Context, agent *agents.Agent, nativeScope, ref string) (PluginRecord, error) {
	if agent.Plugin == nil {
		return PluginRecord{}, &BlockedError{Agent: agent.Name, Reason: ReasonNoPlugin}
	}

	bin, err := probePluginCLI(ctx, agent)
	if err != nil {
		return PluginRecord{}, err
	}

	// Register the marketplace. We only record InstalledMarketplace (and thus
	// later de-register on uninstall) when the marketplace was absent before and
	// our add succeeded, so we never remove a marketplace another plugin shares.
	// On any uncertainty marketplaceRegistered returns true, keeping us off the
	// de-register path.
	//
	// An empty Source marks a built-in marketplace (e.g. Claude's
	// claude-plugins-official): it is already registered, so we never add or
	// de-register it.
	installedMarketplace := false
	if agent.Plugin.Source != "" {
		alreadyPresent := marketplaceRegistered(ctx, bin, agent.Plugin.Marketplace)
		_, addErr := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, marketplaceAddArgs(agent.Plugin.Source)))
		installedMarketplace = addErr == nil && !alreadyPresent
	}
	if args := marketplaceUpdateArgs(agent); args != nil {
		if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, args)); err != nil {
			if installedMarketplace {
				if _, rmErr := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, marketplaceRemoveArgs(agent.Plugin))); rmErr != nil {
					log.Warnf(ctx, "%s plugin marketplace refresh failed and the marketplace could not be de-registered: %v", agent.DisplayName, rmErr)
				}
			}
			return PluginRecord{}, &BlockedError{Agent: agent.Name, Reason: ReasonInstallFailed, Detail: stderrOf(err)}
		}
	}

	if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, pluginInstallArgs(agent, nativeScope))); err != nil {
		// The proactive refresh above updates every registered marketplace, so a
		// stale built-in copy is already handled. A user who removed the built-in
		// marketplace entirely surfaces here instead, as "not found in marketplace":
		// the refresh can't fail on a marketplace that is no longer registered, so
		// this install step is where that case appears. Only that failure is
		// repairable by re-adding the marketplace; auth or network errors are not, so
		// builtinMarketplaceRepairable gates the recovery to avoid prompting to re-add
		// a marketplace that is present. The repair re-adds and refreshes shared
		// infrastructure; it never records ownership, so uninstall still leaves the
		// built-in marketplace in place.
		if builtinMarketplaceRepairable(agent, err) {
			if rec, ok := recoverBuiltinMarketplace(ctx, bin, agent, nativeScope, ref, err); ok {
				return rec, nil
			}
			return PluginRecord{}, builtinMarketplaceError(agent, err)
		}
		// Roll back a marketplace we just added so a failed install doesn't
		// leave an orphaned, untracked marketplace registration behind.
		if installedMarketplace {
			if _, rmErr := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, marketplaceRemoveArgs(agent.Plugin))); rmErr != nil {
				log.Warnf(ctx, "%s plugin install failed and the marketplace could not be de-registered: %v", agent.DisplayName, rmErr)
			}
		}
		return PluginRecord{}, &BlockedError{Agent: agent.Name, Reason: ReasonInstallFailed, Detail: stderrOf(err)}
	}

	return PluginRecord{
		Marketplace:          agent.Plugin.Marketplace,
		Plugin:               agent.Plugin.ID,
		Scope:                nativeScope,
		Version:              DisplaySkillsVersion(ref),
		InstalledMarketplace: installedMarketplace,
	}, nil
}

// recoverBuiltinMarketplace handles a built-in-marketplace install failure whose
// error says the marketplace is missing (a user removed it); the proactive
// refresh already covers a merely-stale copy. In an interactive session it asks
// permission, re-adds and refreshes the marketplace, and retries the install
// once. It returns (record, true) only when that retry succeeds. When the
// session is non-interactive, the user declines, or the retry still fails, it
// returns (_, false) so the caller surfaces an actionable error.
//
// The built-in marketplace is shared infrastructure: even after re-adding it we
// never set InstalledMarketplace, so uninstall never de-registers it.
func recoverBuiltinMarketplace(ctx context.Context, bin string, agent *agents.Agent, nativeScope, ref string, installErr error) (PluginRecord, bool) {
	// Only prompt when there is a terminal that can answer. Code that calls
	// InstallPluginForAgent without a cmdio (e.g. some tests) must not panic.
	if !cmdio.HasIO(ctx) || !cmdio.IsPromptSupported(ctx) {
		return PluginRecord{}, false
	}

	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, fmt.Sprintf("%s could not install the databricks plugin from the %q marketplace:", agent.DisplayName, agent.Plugin.Marketplace))
	cmdio.LogString(ctx, "  "+stderrOf(installErr))
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, fmt.Sprintf("The %q marketplace may be missing or out of date. Add and refresh it with:", agent.Plugin.Marketplace))
	cmdio.LogString(ctx, "  "+strings.Join(repairCommands(agent), "\n  "))
	cmdio.LogString(ctx, "")
	proceed, err := cmdio.AskYesOrNo(ctx, "Run these and retry the install?")
	if err != nil || !proceed {
		return PluginRecord{}, false
	}

	// Re-add (fixes a removed marketplace) then refresh (fixes a stale copy). Each
	// step may harmlessly fail (e.g. add when it is already present), so we only
	// log at debug level and let the retried install be the real verdict.
	if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, marketplaceAddArgs(agent.Plugin.BuiltinAddSource))); err != nil {
		log.Debugf(ctx, "re-adding the %s marketplace failed (it may already be present): %v", agent.Plugin.Marketplace, stderrOf(err))
	}
	if args := marketplaceUpdateArgs(agent); args != nil {
		if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, args)); err != nil {
			log.Debugf(ctx, "refreshing the %s marketplace failed: %v", agent.Plugin.Marketplace, stderrOf(err))
		}
	}
	if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, pluginInstallArgs(agent, nativeScope))); err != nil {
		return PluginRecord{}, false
	}

	return PluginRecord{
		Marketplace: agent.Plugin.Marketplace,
		Plugin:      agent.Plugin.ID,
		Scope:       nativeScope,
		Version:     DisplaySkillsVersion(ref),
	}, true
}

// repairCommands returns the re-add and refresh commands shared by the recovery
// prompt and the non-interactive BlockedError, so the two can't drift apart.
func repairCommands(agent *agents.Agent) []string {
	return []string{
		fmt.Sprintf("%s plugin marketplace add %s", agent.Binary, agent.Plugin.BuiltinAddSource),
		fmt.Sprintf("%s plugin marketplace update %s", agent.Binary, agent.Plugin.Marketplace),
	}
}

// builtinMarketplaceError wraps a built-in-marketplace install failure with the
// exact commands that repair it, so a user who wasn't prompted (non-interactive)
// or whose automatic retry failed knows the issue and the fix.
func builtinMarketplaceError(agent *agents.Agent, installErr error) error {
	return &BlockedError{
		Agent:  agent.Name,
		Reason: ReasonInstallFailed,
		Detail: fmt.Sprintf(
			"%s\n\nThe %q marketplace is missing or out of date. Fix it with:\n  %s\nthen re-run the install.",
			stderrOf(installErr),
			agent.Plugin.Marketplace,
			strings.Join(repairCommands(agent), "\n  ")),
	}
}

// UpdatePluginForAgent updates the plugin through the agent's own CLI. The
// plugin's own update handles content the release dropped, so there is no
// per-skill prune for plugin agents.
func UpdatePluginForAgent(ctx context.Context, agent *agents.Agent, rec PluginRecord) error {
	if agent.Plugin == nil {
		return &BlockedError{Agent: agent.Name, Reason: ReasonNoPlugin}
	}
	bin, err := resolveAgentBinary(agent)
	if err != nil {
		return &BlockedError{Agent: agent.Name, Reason: ReasonCLINotOnPath, Detail: err.Error()}
	}
	for _, args := range pluginUpdateSteps(agent, rec) {
		if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, args)); err != nil {
			return &BlockedError{Agent: agent.Name, Reason: ReasonInstallFailed, Detail: stderrOf(err)}
		}
	}
	return nil
}

// UninstallPluginForAgent removes the plugin through the agent's own CLI, and
// de-registers the marketplace only when this CLI registered it and the caller
// did not ask to keep it. It never removes a marketplace another plugin shares.
//
// A returned error means the plugin itself could not be removed, so the caller
// should keep its state record. Once the plugin is removed, a failure to
// de-register the marketplace is only warned about (not returned): the plugin is
// gone, so the record is cleared, and the leftover marketplace registration is
// harmless and can be removed manually.
func UninstallPluginForAgent(ctx context.Context, agent *agents.Agent, rec PluginRecord, keepMarketplace bool) error {
	if agent.Plugin == nil {
		return &BlockedError{Agent: agent.Name, Reason: ReasonNoPlugin}
	}
	bin, err := resolveAgentBinary(agent)
	if err != nil {
		return &BlockedError{Agent: agent.Name, Reason: ReasonCLINotOnPath, Detail: err.Error()}
	}
	if args := pluginDisableArgs(agent, rec); args != nil {
		if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, args)); err != nil && !claudeAlreadyDisabled(err) {
			return &BlockedError{Agent: agent.Name, Reason: ReasonInstallFailed, Detail: stderrOf(err)}
		}
	}
	if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, pluginUninstallArgs(agent, rec))); err != nil {
		return &BlockedError{Agent: agent.Name, Reason: ReasonInstallFailed, Detail: stderrOf(err)}
	}
	// Never de-register a built-in marketplace (empty Source, e.g. Claude's
	// claude-plugins-official): it is shared infrastructure we did not add.
	if rec.InstalledMarketplace && !keepMarketplace && agent.Plugin.Source != "" {
		if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, marketplaceRemoveArgsForRecord(agent, rec))); err != nil {
			log.Warnf(ctx, "Removed the %s plugin but could not de-register its marketplace (remove it manually if needed): %v", agent.DisplayName, stderrOf(err))
		}
	}
	return nil
}

func claudeAlreadyDisabled(err error) bool {
	return strings.Contains(strings.ToLower(stderrOf(err)), "already disabled")
}

// marketplaceMissing reports whether an install failed because the plugin's
// marketplace is missing or stale, as opposed to an auth or network failure. The
// agent CLIs expose no error code for this, so we match the stderr the way
// claudeAlreadyDisabled does. Claude reports this as `Plugin "<id>" not found in
// marketplace "<name>"`. Callers use it only to decide whether re-adding the
// marketplace is worth attempting, never as the final verdict (the retried
// install is).
func marketplaceMissing(err error) bool {
	return strings.Contains(strings.ToLower(stderrOf(err)), "not found in marketplace")
}

// builtinMarketplaceRepairable reports whether a failed refresh or install is the
// removed-built-in-marketplace case recoverBuiltinMarketplace can repair: the
// marketplace has no routine add (empty Source) but a known re-add source, and
// the error says the marketplace is missing rather than an unrelated failure.
func builtinMarketplaceRepairable(agent *agents.Agent, err error) bool {
	return agent.Plugin.Source == "" && agent.Plugin.BuiltinAddSource != "" && marketplaceMissing(err)
}

// RecordPluginInstalls persists plugin install records into the state file for
// the given CLI scope (global or project), creating state if none exists. ref
// is the resolved skills release the install corresponds to.
func RecordPluginInstalls(ctx context.Context, cliScope string, records map[string]PluginRecord, ref string) error {
	dir, err := skillsDir(ctx, cliScope)
	if err != nil {
		return err
	}
	state, err := LoadState(dir)
	if err != nil {
		return err
	}
	if state == nil {
		// Initialize all maps so a later skills install/update can assign into a
		// plugin-only state without hitting a nil map.
		state = &InstallState{
			SchemaVersion: schemaVersionV2,
			Skills:        map[string]string{},
			RepoDirs:      map[string]string{},
			Files:         map[string]FileRecord{},
		}
	}
	if state.Plugins == nil {
		state.Plugins = make(map[string]PluginRecord, len(records))
	}
	maps.Copy(state.Plugins, records)
	state.Release = ref
	state.LastUpdated = time.Now()
	state.Scope = cliScope
	return SaveState(dir, state)
}

// prepend returns a fresh argv with bin as argv[0] followed by args.
func prepend(bin string, args []string) []string {
	argv := make([]string, 0, len(args)+1)
	argv = append(argv, bin)
	return append(argv, args...)
}
