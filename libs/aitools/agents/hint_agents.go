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
	// Set by codex-cli (0.166.0) for every shell tool call, next to CODEX_CI.
	NameCodex: "CODEX_THREAD_ID",
}

// PluginRecordedFunc reports whether the CLI install state records a
// databricks plugin install for the named agent. The installer package owns
// that state and provides it (installer.GlobalPluginRecorded).
type PluginRecordedFunc func(ctx context.Context, agentName string) bool

// agentInstallHint returns the hint for a non-Claude registry agent. The text
// keeps Claude Code's imperative wording (hintText) and names the agent and
// scope, so the install acts on the calling agent and never stops at the agent
// picker or the scope prompt when the agent's shell is a terminal.
//
// The hint is gated on the agent's own tooling (Agent.DatabricksTooling, with
// pluginRecorded as the plugin source for Codex and Copilot). A plugin
// installed with the agent's own CLI rather than `databricks aitools install`
// leaves no trace there for those two, so their users are hinted; the
// throttle bounds that to once per session or per TTL.
func agentInstallHint(a *Agent, pluginRecorded PluginRecordedFunc) *agentHint {
	return &agentHint{
		installed: func(ctx context.Context) bool {
			recorded := func() bool { return pluginRecorded(ctx, a.Name) }
			return a.DatabricksTooling(ctx, recorded) != ToolingNone
		},
		text:             "Databricks skills are not installed. To work with Databricks reliably, first run: databricks aitools install --agents " + a.Name + " --scope global",
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
// these agents get no hint.
func claimHintMarker(ctx context.Context, key string) bool {
	c := cache.NewCache(ctx, hintCacheComponent, hintSessionTTL, nil)
	cache.Put(ctx, c, key, true)
	_, ok := cache.Get[bool](ctx, c, key)
	return ok
}
