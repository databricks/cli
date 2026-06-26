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
	// ReasonManualOnly: the agent has a plugin but no headless install path (Cursor).
	ReasonManualOnly = "manual-only"
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
func pluginUpdateSteps(agent *agents.Agent) [][]string {
	target := installTarget(agent.Plugin)
	switch agent.Name {
	case agents.NameCodex:
		return [][]string{
			{"plugin", "marketplace", "upgrade"},
			{"plugin", "add", target},
		}
	default:
		return [][]string{{"plugin", "update", target}}
	}
}

// pluginUninstallArgs builds the per-agent uninstall argv (sans binary).
// Codex removes with `plugin remove`; the others use `plugin uninstall`.
func pluginUninstallArgs(agent *agents.Agent) []string {
	target := installTarget(agent.Plugin)
	switch agent.Name {
	case agents.NameCodex:
		return []string{"plugin", "remove", target}
	default:
		return []string{"plugin", "uninstall", target}
	}
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
	if agent.Plugin == nil || agent.Plugin.ManualOnly {
		return PluginRecord{}, &BlockedError{Agent: agent.Name, Reason: ReasonManualOnly}
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
	alreadyPresent := marketplaceRegistered(ctx, bin, agent.Plugin.Marketplace)
	_, addErr := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, marketplaceAddArgs(agent.Plugin)))
	installedMarketplace := addErr == nil && !alreadyPresent

	if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, pluginInstallArgs(agent, nativeScope))); err != nil {
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
		Version:              strings.TrimPrefix(ref, "v"),
		InstalledMarketplace: installedMarketplace,
	}, nil
}

// UpdatePluginForAgent updates the plugin through the agent's own CLI. The
// plugin's own update handles content the release dropped, so there is no
// per-skill prune for plugin agents.
func UpdatePluginForAgent(ctx context.Context, agent *agents.Agent) error {
	if agent.Plugin == nil || agent.Plugin.ManualOnly {
		return &BlockedError{Agent: agent.Name, Reason: ReasonManualOnly}
	}
	bin, err := resolveAgentBinary(agent)
	if err != nil {
		return &BlockedError{Agent: agent.Name, Reason: ReasonCLINotOnPath, Detail: err.Error()}
	}
	for _, args := range pluginUpdateSteps(agent) {
		if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, args)); err != nil {
			return &BlockedError{Agent: agent.Name, Reason: ReasonInstallFailed, Detail: stderrOf(err)}
		}
	}
	return nil
}

// UninstallPluginForAgent removes the plugin through the agent's own CLI, and
// de-registers the marketplace only when this CLI registered it and the caller
// did not ask to keep it. It never removes a marketplace another plugin shares.
func UninstallPluginForAgent(ctx context.Context, agent *agents.Agent, rec PluginRecord, keepMarketplace bool) error {
	if agent.Plugin == nil || agent.Plugin.ManualOnly {
		return &BlockedError{Agent: agent.Name, Reason: ReasonManualOnly}
	}
	bin, err := resolveAgentBinary(agent)
	if err != nil {
		return &BlockedError{Agent: agent.Name, Reason: ReasonCLINotOnPath, Detail: err.Error()}
	}
	if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, pluginUninstallArgs(agent))); err != nil {
		return &BlockedError{Agent: agent.Name, Reason: ReasonInstallFailed, Detail: stderrOf(err)}
	}
	if rec.InstalledMarketplace && !keepMarketplace {
		if _, err := runAgentCmd(ctx, pluginCmdTimeout, prepend(bin, marketplaceRemoveArgs(agent.Plugin))); err != nil {
			return fmt.Errorf("removed plugin but failed to de-register marketplace for %s: %w", agent.DisplayName, err)
		}
	}
	return nil
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
