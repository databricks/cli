package agents

import (
	"context"
	"strconv"
	"time"

	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

const (
	// hintText recommends installing the Databricks AI tooling. The wording
	// is deliberately chosen to be effective at helping agents; change it only
	// after thorough testing.
	hintText = "Databricks skills are not installed. To work with Databricks reliably, first run: databricks aitools install"

	// claudeAgentProvider is the useragent.AgentProvider() value for Claude Code.
	// The SDK maps the CLAUDECODE env var to this product name.
	// https://github.com/databricks/databricks-sdk-go/blob/main/useragent/agent.go
	claudeAgentProvider = "claude-code"

	// claudeNativeHint makes Claude Code offer the user a one-click install of
	// the Databricks plugin. Claude Code strips the line before the model sees
	// the output and decides when to prompt: only in an attended interactive
	// session, at most once per session, and once ever per plugin for that
	// Claude Code user config (so in practice once per machine), whatever the
	// answer. It is still throttled with the text hint: Claude Code strips it
	// only from the tool output, so in a `2>&1 | jq` pipeline the next process
	// reads it raw, and after a "No" repeating it can't lead to an install. The
	// format is fixed by Claude Code: https://code.claude.com/docs/en/plugins/cli-hints
	claudeNativeHint = `<claude-code-hint v="1" type="plugin" value="` + databricksPluginID + "@" + claudeOfficialMarketplace + `" />`

	// claudeChildSessionEnv is set only in subprocesses Claude Code itself
	// spawns (v2.1.172+). Unlike CLAUDECODE, IDE extensions don't set it in
	// their integrated terminals, so gating the native hint on it keeps the raw
	// tag away from humans, who would see it unstripped.
	claudeChildSessionEnv = "CLAUDE_CODE_CHILD_SESSION"

	// ciEnv is the generic CI marker. Together with the SDK's CI/CD provider
	// detection it keeps the native hint out of CI: Claude Code never prompts in
	// unattended runs.
	ciEnv = "CI"

	// aiToolsCommandName is the top-level command whose subtree is never hinted:
	// telling the agent to run `databricks aitools install` while it is already
	// running an aitools command makes no sense.
	aiToolsCommandName = "aitools"

	// authCommandName and tokenCommandName name `databricks auth token`, the
	// credential helper the SDKs and Terraform run. They embed its stderr
	// mid-line in their own errors, where Claude Code can't strip the native
	// line and neither hint is addressed to anyone, so it is never hinted.
	authCommandName  = "auth"
	tokenCommandName = "token"

	// dockerCommandName names `databricks auth docker token`, which Docker runs
	// through docker-credential-databricks and treats the same way.
	dockerCommandName = "docker"

	// selfExecEnv marks a CLI process that the CLI itself started (see
	// SelfExecEnv). Its output goes to the parent CLI, which already hinted
	// if a hint was due and may embed that output mid-line in its own errors,
	// where Claude Code can't strip the native line.
	selfExecEnv = "DATABRICKS_CLI_SELF_EXEC"

	// claudeSessionEnv is Claude Code's per-session identifier. It keys the
	// per-session throttle shared by the text and native lines (at most one
	// hint per hintSessionTTL per session); when unset (older Claude) the hint
	// degrades to per-command.
	claudeSessionEnv = "CLAUDE_CODE_SESSION_ID"

	// hintCacheComponent names the cache bucket holding "already hinted"
	// markers (per session, or per agent on this machine).
	hintCacheComponent = "aitools-hint"
)

// hintSessionTTL is how long a session's "already hinted" marker lives. Within
// that window a session is hinted at most once; after it expires the hint may
// fire again for the same session.
const hintSessionTTL = time.Hour

// agentHint describes how one calling agent is nudged to install the
// Databricks AI tooling.
type agentHint struct {
	// installed reports whether the agent already has the tooling; when it
	// does, nothing is printed.
	installed func(ctx context.Context) bool

	// text is the model-facing recommendation, printed at most once per
	// hintSessionTTL per session (see sessionEnv), or per agent on this machine
	// when there is no session id (see fallbackKey), together with native.
	text string

	// textKind tags the hint-shown event logged when text prints.
	textKind protos.AitoolsHintKind

	// sessionEnv names the env var carrying the agent's session id, which keys
	// the throttle for both lines. When it is empty or unset at runtime,
	// fallbackKey keys the throttle instead.
	sessionEnv string

	// sessionKeyPrefix namespaces the session id in the throttle key, so one
	// agent's marker never silences another agent that reuses the same id.
	// Empty for Claude Code, whose key is the raw session id.
	sessionKeyPrefix string

	// fallbackKey keys the throttle when there is no session id, so the hint
	// prints at most once per hintSessionTTL on this machine. Empty means no
	// throttle without a session id (Claude Code).
	fallbackKey string

	// durableThrottle prints the text only once its throttle marker is
	// persisted (see claimHintMarker), so an unwritable cache can't turn the
	// hint into a line on every command. False for
	// Claude Code, whose marker write stays best-effort.
	durableThrottle bool

	// native returns a line the agent itself acts on (for Claude Code, its
	// plugin install prompt), or "" when none applies to this invocation. It
	// shares the text's per-session throttle. Nil for agents without one.
	native func(ctx context.Context) string

	// nativeKind tags the hint-shown event logged when the native line prints.
	nativeKind protos.AitoolsHintKind
}

// hintForAgent returns the hint for an agent, keyed by its
// useragent.AgentProvider() value, or nil when that agent is not hinted.
func hintForAgent(provider string, pluginRecorded PluginRecordedFunc) *agentHint {
	switch provider {
	case claudeAgentProvider:
		return &agentHint{
			installed:  claudeToolingInstalled,
			text:       hintText,
			textKind:   protos.AitoolsHintKindInstallText,
			sessionEnv: claudeSessionEnv,
			native:     claudeNativeHintLine,
			nativeKind: protos.AitoolsHintKindClaudeCodeNative,
		}
	default:
		if a := AgentForProvider(provider); a != nil {
			return agentInstallHint(a, pluginRecorded)
		}
		return nil
	}
}

// nativeLine returns the agent-native line due for this invocation, or "".
func (h *agentHint) nativeLine(ctx context.Context) string {
	if h.native == nil {
		return ""
	}
	return h.native(ctx)
}

// MaybeHint recommends installing the Databricks AI tooling when a supported
// agent drives the CLI without it installed. It is a no-op for human callers,
// unsupported agents, aitools commands themselves, the `auth token` and `auth
// docker token` credential helpers, a CLI process the CLI started itself, when
// there is no output stream, and once the tooling is present.
// Both lines, the agent-native one (if any) and the model-facing text, and
// their hint-shown events are throttled together per session, or per agent on
// this machine for agents without a session id (fallbackKey).
// pluginRecorded reads the CLI install record for agents whose plugin has no
// manifest reader here (see Agent.DatabricksTooling).
// Best-effort: it never fails the command. Besides the stderr lines, it logs a
// telemetry event for each hint line shown, which makes a command that resolved
// auth upload telemetry at exit.
func MaybeHint(ctx context.Context, cmd *cobra.Command, pluginRecorded PluginRecordedFunc) {
	hint := hintForAgent(useragent.AgentProvider(), pluginRecorded)
	if hint == nil {
		return
	}
	if isAIToolsCommand(cmd) {
		return
	}
	if isCredentialHelperCommand(cmd) {
		return
	}
	if env.Get(ctx, selfExecEnv) != "" {
		return
	}
	if !cmdio.HasIO(ctx) {
		return
	}
	// Check the throttle before the filesystem probe, so a throttled session
	// skips the plugin/skills detection. The marker is written only once a hint
	// is due, so an already-installed environment never burns it.
	key := hint.throttleKey(ctx)
	if alreadyHintedThisSession(ctx, key) {
		return
	}
	if hint.installed(ctx) {
		return
	}
	if !hint.claim(ctx, key) {
		return
	}
	if native := hint.nativeLine(ctx); native != "" {
		cmdio.LogString(ctx, native)
		logHintShown(ctx, hint.nativeKind)
	}
	cmdio.LogString(ctx, hint.text)
	logHintShown(ctx, hint.textKind)
}

// claim records the throttle key and reports whether the hint may print. A
// durable throttle allows it only once the marker has persisted.
func (h *agentHint) claim(ctx context.Context, key string) bool {
	if h.durableThrottle {
		return claimHintMarker(ctx, key)
	}
	recordSessionHint(ctx, key)
	return true
}

// claudeNativeHintLine returns claudeNativeHint when Claude Code itself spawned
// this process outside CI and no standard stream is a terminal, and ""
// otherwise.
//
// Claude Code runs its tools with no terminal (stdin from /dev/null, output
// captured) and strips the tag from the captured output. tmux, screen or a
// shell started from inside Claude Code inherit claudeChildSessionEnv, and
// nothing strips the tag there. A person typing in one has the terminal on
// stdin even when the output goes through a pipe (`2>&1 | less`), so any
// terminal means a human may read the tag raw.
func claudeNativeHintLine(ctx context.Context) string {
	if env.Get(ctx, claudeChildSessionEnv) == "" || inCI(ctx) {
		return ""
	}
	if cmdio.HasTerminal(ctx) {
		return ""
	}
	return claudeNativeHint
}

// inCI reports whether the CLI runs in CI: the generic CI variable set to
// anything but an explicit false (some providers set it to their own name), or
// any provider the SDK detects for its user agent (e.g. Azure Pipelines'
// TF_BUILD, which doesn't set CI).
func inCI(ctx context.Context) bool {
	if v := env.Get(ctx, ciEnv); v != "" {
		if b, err := strconv.ParseBool(v); err != nil || b {
			return true
		}
	}
	return useragent.CiCdProvider() != ""
}

// isCredentialHelperCommand reports whether cmd is `databricks auth token` or
// `databricks auth docker token`.
func isCredentialHelperCommand(cmd *cobra.Command) bool {
	if cmd.Name() != tokenCommandName {
		return false
	}
	parent := cmd.Parent()
	if parent != nil && parent.Name() == dockerCommandName {
		parent = parent.Parent()
	}
	return parent != nil && parent.Name() == authCommandName
}

// isAIToolsCommand reports whether cmd is the aitools command or one of its
// subcommands, walking the parent chain (mirrors versioncheck.isExemptCommand).
func isAIToolsCommand(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == aiToolsCommandName {
			return true
		}
	}
	return false
}

// claudeToolingInstalled reports whether Claude Code has Databricks AI tooling
// in its own locations (see Agent.DatabricksTooling): the databricks plugin, or
// databricks skills in Claude's skills dirs. Skills present only in the shared
// ~/.databricks/aitools/skills store don't count, because Claude Code doesn't
// load them from there. Any read/parse failure reports false, so an unreadable
// state errs toward hinting.
func claudeToolingInstalled(ctx context.Context) bool {
	claude := ByName(NameClaudeCode)
	if claude == nil {
		// claude-code is always in the registry; this guard keeps the hint
		// best-effort rather than panicking in a persistent pre-run hook.
		return false
	}
	// Claude Code's own manifest decides, so the CLI's install record is never
	// consulted for it.
	tooling := claude.DatabricksTooling(ctx, func() bool { return false })
	if tooling == ToolingPlugin {
		version, _ := claude.DatabricksPluginVersion(ctx)
		log.Debugf(ctx, "Databricks plugin detected for Claude Code (version %q); skipping install hint", version)
	}
	return tooling != ToolingNone
}

// throttleKey returns the cache key throttling the hint: the session id behind
// sessionKeyPrefix (Claude's key is the raw id and must not change), else
// fallbackKey. "" means no throttle.
func (h *agentHint) throttleKey(ctx context.Context) string {
	if session := sessionID(ctx, h.sessionEnv); session != "" {
		return h.sessionKeyPrefix + session
	}
	return h.fallbackKey
}

// alreadyHintedThisSession reports whether the throttle key was hinted within
// the last hintSessionTTL. With no key it returns false so the hint fires per
// command.
func alreadyHintedThisSession(ctx context.Context, key string) bool {
	if key == "" {
		return false
	}
	c := cache.NewCache(ctx, hintCacheComponent, hintSessionTTL, nil)
	_, seen := cache.Get[bool](ctx, c, key)
	return seen
}

// recordSessionHint marks the throttle key as hinted, so later commands stay
// quiet until the marker expires (hintSessionTTL). No-op when the key is "".
func recordSessionHint(ctx context.Context, key string) {
	if key == "" {
		return
	}
	c := cache.NewCache(ctx, hintCacheComponent, hintSessionTTL, nil)
	cache.Put(ctx, c, key, true)
}

// sessionID returns the value of sessionEnv, or "" when the agent has no
// session env var or it is unset.
func sessionID(ctx context.Context, sessionEnv string) string {
	if sessionEnv == "" {
		return ""
	}
	return env.Get(ctx, sessionEnv)
}

// SelfExecEnv returns environ plus the marker that keeps a CLI process the
// CLI starts of itself from hinting. Use it as the child's exec.Cmd.Env.
func SelfExecEnv(environ []string) []string {
	return append(environ, selfExecEnv+"=1")
}

// logHintShown records that a recommendation to install Databricks AI tools of
// the given kind was shown. Call it once per hint shown; for a prompt, call it
// before asking so a declined prompt still counts. The upload's
// ExecutionContext.Command names the command the hint was shown on.
//
// The event is buffered and uploaded by cmd/root at exit, which needs resolved
// auth: a hint shown on a command that never resolves auth (e.g. `databricks
// version`) is not uploaded.
func logHintShown(ctx context.Context, kind protos.AitoolsHintKind) {
	telemetry.Log(ctx, protos.DatabricksCliLog{
		AitoolsHintEvent: &protos.AitoolsHintEvent{HintKind: kind},
	})
}
