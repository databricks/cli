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
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
)

// lookPath resolves a binary on PATH. It is a package-level var so tests can
// inject a fake resolver without depending on the host PATH.
var lookPath = exec.LookPath

const (
	// pluginProbeTimeout bounds the `<agent> plugin --help` capability check.
	// Agent CLIs are large Node/Rust binaries whose cold start (first run,
	// antivirus scan, slow container disk) can take several seconds, and a probe
	// that times out reports the agent as unavailable, so leave headroom.
	// It also bounds `<agent> plugin marketplace list` in marketplaceRegistered,
	// where a timeout means the marketplace is assumed already registered, so
	// install does not record it and uninstall leaves it registered.
	pluginProbeTimeout = 20 * time.Second
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
	// TimedOut is set when the agent CLI was killed for exceeding its timeout,
	// so the failure can be told apart from a missing or broken CLI. It takes
	// precedence over Reason: Reason still names the step that timed out, so
	// check TimedOut before switching on Reason.
	TimedOut bool
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
// error. Errors wrap a *process.ProcessError, which carries the captured stderr;
// on timeout the error is an *agentCmdTimeoutError that wraps it, so callers
// must use errors.AsType rather than a type assertion.
func runAgentCmd(ctx context.Context, timeout time.Duration, argv []string) (string, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out, err := process.Background(cctx, argv)
	if err != nil && errors.Is(cctx.Err(), context.DeadlineExceeded) {
		return out, &agentCmdTimeoutError{timeout: timeout, err: err}
	}
	return out, err
}

// agentCmdTimeoutError reports an agent CLI command killed by its timeout.
type agentCmdTimeoutError struct {
	timeout time.Duration
	err     error
}

func (e *agentCmdTimeoutError) Error() string {
	return fmt.Sprintf("timed out after %s", e.timeout)
}

func (e *agentCmdTimeoutError) Unwrap() error { return e.err }

// newBlockedError builds the BlockedError for a failed agent CLI command.
func newBlockedError(agent *agents.Agent, reason string, err error) *BlockedError {
	_, timedOut := errors.AsType[*agentCmdTimeoutError](err)
	return &BlockedError{Agent: agent.Name, Reason: reason, Detail: stderrOf(err), TimedOut: timedOut}
}

// stderrOf returns the captured stderr of a failed agent command, falling back
// to the error's own message. Callers must not branch on this string.
func stderrOf(err error) string {
	var stderr string
	if perr, ok := errors.AsType[*process.ProcessError](err); ok {
		stderr = strings.TrimSpace(perr.Stderr)
	}
	// The timeout is the real cause of a killed process. Whatever it wrote to
	// stderr before the kill (a prompt waiting for input, say) follows it.
	if terr, ok := errors.AsType[*agentCmdTimeoutError](err); ok {
		if stderr != "" {
			return terr.Error() + ": " + stderr
		}
		return terr.Error()
	}
	if stderr != "" {
		return stderr
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
	return plugin + "@" + recordedMarketplace(agent, rec)
}

// marketplaceAddArgs builds the `plugin marketplace add <source>` argv (sans binary).
func marketplaceAddArgs(spec *agents.PluginSpec) []string {
	return []string{"plugin", "marketplace", "add", spec.Source}
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
	return []string{"plugin", "marketplace", "remove", recordedMarketplace(agent, rec)}
}

// recordedMarketplace is the marketplace a recorded install came from, or the
// agent's current one for records written before it was stored.
func recordedMarketplace(agent *agents.Agent, rec PluginRecord) string {
	if rec.Marketplace != "" {
		return rec.Marketplace
	}
	return agent.Plugin.Marketplace
}

// marketplaceUpdateArgs builds the marketplace refresh command for agents whose
// plugin CLI supports it. Claude's official marketplace can be stale locally,
// causing install/update to report that a published plugin does not exist.
// The refresh names the marketplace: without a name Claude refreshes every
// marketplace the user has added, and one unreachable third-party marketplace
// then fails the databricks install.
func marketplaceUpdateArgs(agent *agents.Agent, marketplace string) []string {
	if agent.Name == agents.NameClaudeCode {
		return []string{"plugin", "marketplace", "update", marketplace}
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

// pluginRefreshArgs builds the marketplace refresh that runs before an update
// (sans binary), or nil for agents without one. Both agents name the
// marketplace, so an unreachable marketplace the user added for another plugin
// does not fail the update.
func pluginRefreshArgs(agent *agents.Agent, rec PluginRecord) []string {
	if agent.Name == agents.NameCodex {
		return []string{"plugin", "marketplace", "upgrade", recordedMarketplace(agent, rec)}
	}
	return marketplaceUpdateArgs(agent, recordedMarketplace(agent, rec))
}

// pluginUpdateArgs builds the per-agent update argv (sans binary). Codex
// updates by re-adding the plugin.
func pluginUpdateArgs(agent *agents.Agent, rec PluginRecord) []string {
	target := recordedInstallTarget(agent, rec)
	switch agent.Name {
	case agents.NameCodex:
		return []string{"plugin", "add", target}
	case agents.NameClaudeCode:
		return appendScopeArg([]string{"plugin", "update", target}, rec.Scope)
	default:
		return []string{"plugin", "update", target}
	}
}

// refreshMarketplace runs the marketplace refresh (args may be nil for agents
// without one). A failed refresh is not fatal: the local copy of the marketplace
// usually still lists the plugin, so the caller goes on with it, and either
// warns with warnLocalMarketplaceCopy when that succeeds and the recorded
// version is cleared, or adds the refresh error to its own error with
// withRefreshError.
func refreshMarketplace(ctx context.Context, bin string, args []string) error {
	if args == nil {
		return nil
	}
	_, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, args))
	return err
}

// warnLocalMarketplaceCopy tells the user that a successful install or update
// used a local marketplace copy that could not be refreshed, so the plugin may
// be out of date. retry names what to run later; it differs between install
// and update so an update is not told to run the command that just ran.
func warnLocalMarketplaceCopy(ctx context.Context, agent *agents.Agent, marketplace, retry string, refreshErr error) {
	log.Warnf(ctx, "Could not refresh the %s marketplace, so the %s plugin came from its local copy and may be out of date; %s: %s", marketplace, agent.DisplayName, retry, stderrOf(refreshErr))
}

// withRefreshError appends a failed refresh to the detail of a failed install
// or update, since a stale local marketplace copy may be the cause.
func withRefreshError(blocked *BlockedError, refreshErr error) *BlockedError {
	if refreshErr != nil {
		blocked.Detail += " (refreshing the marketplace also failed: " + stderrOf(refreshErr) + ")"
	}
	return blocked
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
		return "", newBlockedError(agent, ReasonCLINotOnPath, err)
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
	installedMarketplace := false
	if agent.Plugin.Source != "" {
		alreadyPresent := marketplaceRegistered(ctx, bin, agent.Plugin.Marketplace)
		_, addErr := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, marketplaceAddArgs(agent.Plugin)))
		installedMarketplace = addErr == nil && !alreadyPresent
	}
	refreshErr := refreshMarketplace(ctx, bin, marketplaceUpdateArgs(agent, agent.Plugin.Marketplace))

	if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, pluginInstallArgs(agent, nativeScope))); err != nil {
		// Roll back a marketplace we just added so a failed install doesn't
		// leave an orphaned, untracked marketplace registration behind.
		if installedMarketplace {
			if _, rmErr := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, marketplaceRemoveArgs(agent.Plugin))); rmErr != nil {
				log.Warnf(ctx, "%s plugin install failed and the marketplace could not be de-registered: %v", agent.DisplayName, rmErr)
			}
		}
		return PluginRecord{}, withRefreshError(newBlockedError(agent, ReasonInstallFailed, err), refreshErr)
	}

	// A marketplace we just added is a fresh copy, so a failed refresh right
	// after it changes nothing. Otherwise we can't tell which version the local
	// copy held, so the record says "version unknown" until the next successful
	// update.
	version := DisplaySkillsVersion(ref)
	if refreshErr != nil && !installedMarketplace {
		warnLocalMarketplaceCopy(ctx, agent, agent.Plugin.Marketplace, "run `databricks aitools update` later", refreshErr)
		version = ""
	}
	return PluginRecord{
		Marketplace:          agent.Plugin.Marketplace,
		Plugin:               agent.Plugin.ID,
		Scope:                nativeScope,
		Version:              version,
		InstalledMarketplace: installedMarketplace,
	}, nil
}

// UpdatePluginForAgent updates the plugin through the agent's own CLI. The
// plugin's own update handles content the release dropped, so there is no
// per-skill prune for plugin agents. When the update succeeded after a failed
// marketplace refresh, it returns that refresh error: the update ran from the
// local copy, and the caller decides whether the recorded version still holds
// and whether to warn.
func UpdatePluginForAgent(ctx context.Context, agent *agents.Agent, rec PluginRecord) (refreshErr, err error) {
	if agent.Plugin == nil {
		return nil, &BlockedError{Agent: agent.Name, Reason: ReasonNoPlugin}
	}
	bin, err := resolveAgentBinary(agent)
	if err != nil {
		return nil, &BlockedError{Agent: agent.Name, Reason: ReasonCLINotOnPath, Detail: err.Error()}
	}
	refreshErr = refreshMarketplace(ctx, bin, pluginRefreshArgs(agent, rec))
	if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, pluginUpdateArgs(agent, rec))); err != nil {
		return nil, withRefreshError(newBlockedError(agent, ReasonInstallFailed, err), refreshErr)
	}
	return refreshErr, nil
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
			return newBlockedError(agent, ReasonInstallFailed, err)
		}
	}
	if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, pluginUninstallArgs(agent, rec))); err != nil {
		return newBlockedError(agent, ReasonInstallFailed, err)
	}
	// Never de-register a Shared marketplace, it is shared infrastructure we don't own
	if rec.InstalledMarketplace && !keepMarketplace && !agent.Plugin.Shared {
		if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, marketplaceRemoveArgsForRecord(agent, rec))); err != nil {
			log.Warnf(ctx, "Removed the %s plugin but could not de-register its marketplace (remove it manually if needed): %v", agent.DisplayName, stderrOf(err))
		}
	}
	return nil
}

func claudeAlreadyDisabled(err error) bool {
	return strings.Contains(strings.ToLower(stderrOf(err)), "already disabled")
}

// RecordPluginInstalls persists plugin install records into the state file for
// the given CLI scope (global or project), creating state if none exists. ref
// is the resolved skills release the install corresponds to. An empty ref
// (release unresolved) keeps the existing state.Release.
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
	// An unresolved release keeps the one recorded by earlier installs.
	if ref != "" {
		state.Release = ref
	}
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
