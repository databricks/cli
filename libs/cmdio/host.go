package cmdio

import (
	"context"

	"github.com/databricks/cli/libs/env"
)

// Host describes the terminal or IDE the CLI is being invoked from.
// Values are an enum, never raw env values, so they are safe to log.
type Host string

const (
	HostVSCode        Host = "vscode"
	HostVSCodeCopilot Host = "vscode-copilot"
	HostCursor        Host = "cursor"
	HostWindsurf      Host = "windsurf"
	HostJetBrains     Host = "jetbrains"
	HostZed           Host = "zed"
	HostWarp          Host = "warp"
	HostITerm         Host = "iterm"
	HostAppleTerminal Host = "apple-terminal"
	HostGhostty       Host = "ghostty"
	HostWezTerm       Host = "wezterm"
	HostHyper         Host = "hyper"
	HostTabby         Host = "tabby"
	HostUnknown       Host = "unknown"
)

// Environment variables we inspect. Sources:
//   - VSCode: https://github.com/microsoft/vscode/blob/main/src/vs/workbench/contrib/terminal/common/terminalEnvironment.ts
//   - Cursor: VSCode fork; sets CURSOR_TRACE_ID in addition to VSCode vars
//   - JetBrains: https://www.jetbrains.com/help/idea/terminal.html
//   - Apple Terminal / iTerm / Warp / Ghostty / WezTerm / Hyper / Tabby: set TERM_PROGRAM directly
const (
	envTermProgram       = "TERM_PROGRAM"
	envTerminalEmulator  = "TERMINAL_EMULATOR"
	envCursorTraceID     = "CURSOR_TRACE_ID"
	envCFBundleID        = "__CFBundleIdentifier"
	envCopilotAgent      = "GITHUB_COPILOT_AGENT_VERSION"
	envCopilotIntegrator = "COPILOT_AGENT_INTEGRATION_ID"
)

// DetectHost returns the terminal or IDE host the CLI is being run from,
// derived from environment variables only. Returns HostUnknown if no
// signals match (the common case for raw shells without TERM_PROGRAM set).
func DetectHost(ctx context.Context) Host {
	// Cursor and Windsurf are VSCode forks: they inherit TERM_PROGRAM=vscode,
	// so check their discriminators before falling through to plain VSCode.
	if env.Get(ctx, envCursorTraceID) != "" {
		return HostCursor
	}
	if env.Get(ctx, envCFBundleID) == "com.exafunction.windsurf" {
		return HostWindsurf
	}

	switch env.Get(ctx, envTermProgram) {
	case "vscode":
		// Best-effort sentinel for invocations driven by VSCode's Copilot
		// coding agent. The exact env vars Copilot sets in agent-mode
		// terminals are not stable yet; treat this as a coarse signal to
		// be refined once we see real telemetry.
		if isCopilotAgent(ctx) {
			return HostVSCodeCopilot
		}
		return HostVSCode
	case "Apple_Terminal":
		return HostAppleTerminal
	case "iTerm.app":
		return HostITerm
	case "WarpTerminal":
		return HostWarp
	case "ghostty":
		return HostGhostty
	case "WezTerm":
		return HostWezTerm
	case "Hyper":
		return HostHyper
	case "Tabby":
		return HostTabby
	case "zed":
		return HostZed
	}

	if env.Get(ctx, envTerminalEmulator) == "JetBrains-JediTerm" {
		return HostJetBrains
	}

	return HostUnknown
}

func isCopilotAgent(ctx context.Context) bool {
	return env.Get(ctx, envCopilotAgent) != "" || env.Get(ctx, envCopilotIntegrator) != ""
}
