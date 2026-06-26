package agents

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/databricks/cli/libs/process"
)

// lookPath resolves a binary on PATH. It is a package-level var so tests can
// inject a fake resolver without depending on the host PATH. It mirrors the
// exec.LookPath precedent in libs/python/detect.go.
var lookPath = exec.LookPath

// pluginProbeTimeout bounds the `<agent> plugin --help` capability probe so a
// hung agent CLI cannot stall install.
const pluginProbeTimeout = 5 * time.Second

// DisplayState describes how an agent should be presented in the install
// picker. It combines the two cheap presence signals (binary on PATH and
// config dir on disk) without running the agent.
type DisplayState string

const (
	// StateAvailable: the agent's CLI binary is on PATH, so its plugin can be installed.
	StateAvailable DisplayState = "available"
	// StateInstalledCLIMissing: the config dir exists but the CLI binary is not on PATH.
	StateInstalledCLIMissing DisplayState = "installed-cli-missing"
	// StateManualOnly: the agent has a plugin but no headless install path (Cursor).
	StateManualOnly DisplayState = "manual-only"
	// StateFilesOnly: the agent has no plugin; skill files are its only delivery.
	StateFilesOnly DisplayState = "files-only"
	// StateNotFound: neither presence signal is present.
	StateNotFound DisplayState = "not-found"
)

// ErrPluginUnsupported is returned by ProbePlugin when the agent cannot install
// the databricks plugin: it has no binary, no plugin, or its CLI does not
// expose a working `plugin` subcommand.
var ErrPluginUnsupported = errors.New("agent CLI does not support plugins")

// HasBinary reports whether the agent's CLI binary is resolvable on PATH.
// Agents with no CLI binary (Antigravity) always report false.
func (a *Agent) HasBinary(_ context.Context) bool {
	if a.Binary == "" {
		return false
	}
	_, err := lookPath(a.Binary)
	return err == nil
}

// DisplayState returns the picker presentation state for the agent. It never
// runs the agent (that is ProbePlugin's job); it only checks the binary on
// PATH and the config dir on disk, so it is cheap enough for every render.
func (a *Agent) DisplayState(ctx context.Context) DisplayState {
	if a.Plugin != nil && a.Plugin.ManualOnly {
		return StateManualOnly
	}

	hasBinary := a.HasBinary(ctx)
	hasConfig := a.Detected(ctx)

	if a.Plugin == nil {
		if hasBinary || hasConfig {
			return StateFilesOnly
		}
		return StateNotFound
	}

	switch {
	case hasBinary:
		return StateAvailable
	case hasConfig:
		return StateInstalledCLIMissing
	default:
		return StateNotFound
	}
}

// Preselect reports whether the agent should be pre-checked in the picker. Only
// agents that can complete an install automatically (a plugin agent with its
// binary on PATH) and files-only agents whose config dir exists are pre-checked;
// manual-only and binary-missing agents are shown but left unchecked.
func (a *Agent) Preselect(ctx context.Context) bool {
	switch a.DisplayState(ctx) {
	case StateAvailable:
		return true
	case StateFilesOnly:
		return a.Detected(ctx)
	default:
		return false
	}
}

// ProbePlugin checks whether the agent's CLI actually supports the plugin
// subcommand by running `<agent> plugin --help` with a short timeout. It is the
// only function here that executes the agent and is meant to be called once,
// for a selected agent, right before install. It returns ErrPluginUnsupported
// when the agent has no plugin path or its CLI lacks the subcommand.
func (a *Agent) ProbePlugin(ctx context.Context) error {
	if a.Binary == "" || a.Plugin == nil || a.Plugin.ManualOnly {
		return ErrPluginUnsupported
	}

	// exec.LookPath returns exec.ErrDot (along with the path) when the binary
	// resolves only relative to the current directory. Returning on any error,
	// including ErrDot, means we never execute a binary resolved from cwd (a
	// malicious ./claude dropped there is refused, not run).
	path, err := lookPath(a.Binary)
	if err != nil {
		return fmt.Errorf("could not resolve %s on PATH: %w", a.Binary, err)
	}

	cctx, cancel := context.WithTimeout(ctx, pluginProbeTimeout)
	defer cancel()

	if _, err := process.Background(cctx, []string{path, "plugin", "--help"}); err != nil {
		return fmt.Errorf("%s plugin subcommand unavailable: %w", a.Binary, errors.Join(ErrPluginUnsupported, err))
	}
	return nil
}
