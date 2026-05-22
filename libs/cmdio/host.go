package cmdio

import (
	"context"

	"github.com/databricks/cli/libs/env"
)

// Host describes the terminal or IDE the CLI is being invoked from.
// Values are an enum, never raw env values, so they are safe to log.
type Host string

const (
	// HostVSCode covers TERM_PROGRAM=vscode, which is set by vanilla VSCode
	// and every fork that inherits its terminal integration (Cursor, Windsurf,
	// code-server, etc.). The forks don't expose a stable, trustworthy
	// discriminator in env, so we deliberately don't try to split them apart.
	HostVSCode Host = "vscode"

	HostJetBrains     Host = "jetbrains"
	HostAppleTerminal Host = "apple-terminal"
	HostITerm         Host = "iterm"
	HostWarp          Host = "warp"
	HostWezTerm       Host = "wezterm"
	HostGhostty       Host = "ghostty"
	HostUnknown       Host = "unknown"
)

const (
	envTermProgram      = "TERM_PROGRAM"
	envTerminalEmulator = "TERMINAL_EMULATOR"
)

// DetectHost returns the terminal or IDE host the CLI is being run from,
// derived from environment variables only.
//
// Only detections backed by direct observation or upstream documentation
// are included. Anything we can't verify (Windsurf vs. Cursor split, Zed,
// Hyper, Tabby, etc.) falls into HostUnknown until we see real evidence.
//
// Whether a user has a particular extension or AI agent active (Copilot,
// Claude Code, Cursor Agent, etc.) is intentionally not modelled here.
// That's an independent dimension, so a downstream query can ask "vscode
// users without Copilot" by joining the two signals.
func DetectHost(ctx context.Context) Host {
	switch env.Get(ctx, envTermProgram) {
	case "vscode":
		return HostVSCode
	case "Apple_Terminal":
		return HostAppleTerminal
	case "iTerm.app":
		return HostITerm
	case "WarpTerminal":
		return HostWarp
	case "WezTerm":
		return HostWezTerm
	case "ghostty":
		return HostGhostty
	}

	// JediTerm is JetBrains' terminal library; sets TERMINAL_EMULATOR
	// per https://github.com/JetBrains/jediterm/issues/253.
	if env.Get(ctx, envTerminalEmulator) == "JetBrains-JediTerm" {
		return HostJetBrains
	}

	return HostUnknown
}
