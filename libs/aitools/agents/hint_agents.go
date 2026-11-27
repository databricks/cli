package agents

import (
	"context"

	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/telemetry/protos"
)

// agentSessionEnv names the env var a non-Claude agent sets to a per-session id
// in the shells it spawns. Only verified variables are listed; an agent missing
// here is throttled per machine instead (see agentInstallHint).
var agentSessionEnv = map[string]string{
	// Set by codex-cli (0.166.0) for every shell tool call, next to CODEX_CI:
	// https://github.com/openai/codex/blob/main/codex-rs/core/src/exec_env.rs
	NameCodex: "CODEX_THREAD_ID",
}

// PluginRecordedFunc reports whether the CLI install state records a
// databricks plugin install for the named agent. The installer package owns
// that state and provides it (installer.PluginRecorded for the global scope).
type PluginRecordedFunc func(ctx context.Context, agentName string) bool

// agentInstallHint returns the hint for a non-Claude registry agent. The text
// is hintText plus --agents and --scope. An agent caller never sees install's
// prompts (promptsSupported), so --agents is what matters: it targets
// the calling agent even when it isn't detected on this machine, instead of
// every detected one. --scope global only makes the non-interactive default
// explicit.
//
// The hint is gated on the agent's own tooling (Agent.DatabricksTooling, which
// for Codex and Copilot reads their own config and pluginRecorded). A plugin
// installed outside `databricks aitools install` in a way that leaves no trace
// there (e.g. from Cursor itself) gets those users hinted; the throttle bounds
// that to once per session or per TTL.
func agentInstallHint(a *Agent, pluginRecorded PluginRecordedFunc) *agentHint {
	return &agentHint{
		agentName: a.Name,
		installed: func(ctx context.Context) bool {
			recorded := func() bool { return pluginRecorded(ctx, a.Name) }
			return a.DatabricksTooling(ctx, recorded) != ToolingNone
		},
		text:             hintText + " --agents " + a.Name + " --scope global",
		textKind:         protos.AitoolsHintKindAgentInstallText,
		sessionEnv:       agentSessionEnv[a.Name],
		sessionKeyPrefix: a.Name + ":session:",
		fallbackKey:      a.Name + ":machine",
		durableThrottle:  true,
	}
}

// claimHintMarker writes the throttle marker to the user cache and reports
// whether it reads back. libs/cache silently drops writes it can't make (e.g.
// a read-only home in an agent sandbox), and without a kept marker the hint
// would repeat on every command, so the caller then skips it. A sandbox that
// can't write the user cache can't run the install either. The same holds with
// caching turned off (DATABRICKS_CACHE_ENABLED=false): nothing persists, so
// these agents get no hint. throttle is the text's throttle (see message).
func claimHintMarker(ctx context.Context, throttle hintThrottle, key string) bool {
	c := throttle.markers(ctx)
	cache.Put(ctx, c, key, true)
	_, ok := cache.Get[bool](ctx, c, key)
	return ok
}
