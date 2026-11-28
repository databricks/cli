package agents

import (
	"context"
	"strings"
	"time"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/skills"
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

	// skillsHintText replaces hintText when skills.HintsEnabled is true. Agents
	// acknowledged the install text without acting on it, and reading a skill
	// through `databricks skills` needs no install step, so it asks for that.
	skillsHintText = "Before you work with Databricks, run: databricks skills list"

	// claudeAgentProvider is the useragent.AgentProvider() value for Claude Code.
	// The SDK maps the CLAUDECODE env var to this product name.
	// https://github.com/databricks/databricks-sdk-go/blob/main/useragent/agent.go
	claudeAgentProvider = "claude-code"

	// claudeNativeHint makes Claude Code offer the user a one-click install of
	// the Databricks plugin. Claude Code strips the line before the model sees
	// the output and decides when to prompt: only in an attended interactive
	// session, at most once per session, and once ever per plugin for that
	// Claude Code user config (so in practice once per machine), whatever the
	// answer. It is still throttled per session (see nativeHintThrottle): Claude
	// Code strips it only from the tool output, so in a `2>&1 | jq` pipeline the
	// next process reads it raw, and after a "No" repeating it can't lead to an
	// install. The format is fixed by Claude Code:
	// https://code.claude.com/docs/en/plugins/cli-hints
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
	// per-session throttles of the text and native lines; when unset (older
	// Claude) both lines degrade to per-command.
	claudeSessionEnv = "CLAUDE_CODE_SESSION_ID"
)

// hintThrottle is an "already hinted" marker in the file cache, per throttle
// key (see agentHint.throttleKey): a line it guards prints at most once per
// ttl per key.
type hintThrottle struct {
	// component names the cache bucket holding the markers. Each throttle needs
	// its own: a cache expires every file in its bucket by its own ttl.
	component string
	ttl       time.Duration
}

var (
	// textHintThrottle throttles the model-facing text hint, which the model
	// reads in every tool output it lands in.
	textHintThrottle = hintThrottle{component: "aitools-hint", ttl: time.Hour}

	// skillsTextHintThrottle throttles the skills list text on its own marker
	// bucket. An install-text marker left within the hour must not hide the
	// skills text when skill hints are turned on, and the reverse.
	skillsTextHintThrottle = hintThrottle{component: "aitools-skills-hint", ttl: time.Hour}

	// nativeHintThrottle throttles the agent-native line on its own, shorter
	// clock. The session id doesn't tell the main loop from a subagent or a
	// hook (a subagent's commands carry the parent's session id and the
	// child-session marker), and Claude Code strips the line there without
	// prompting; with the text hint's hour, a subagent making the session's
	// first call would hide the dialog from the main loop for that hour.
	// Claude Code prompts at most once per session anyway, so repeating the
	// line costs only a raw tag in a `2>&1` pipeline every ten minutes, and a
	// session waits at most that long for the dialog.
	nativeHintThrottle = hintThrottle{component: "aitools-native-hint", ttl: 10 * time.Minute}

	// nativeEventThrottle keeps the native line's hint-shown event on the text
	// hint's clock, so a native-only repeat triggers no telemetry upload at
	// exit and native-event counts don't grow with session length. Its marker
	// holds whether that event was logged in the current text window: each
	// text print starts a window and rewrites it.
	nativeEventThrottle = hintThrottle{component: "aitools-native-hint-event", ttl: textHintThrottle.ttl}
)

// agentHint describes how one calling agent is nudged to install the
// Databricks AI tooling.
type agentHint struct {
	// agentName is the registry name of the hinted agent.
	agentName string

	// installed reports whether the agent already has the tooling; when it
	// does, nothing is printed.
	installed func(ctx context.Context) bool

	// text is the agent's install recommendation, the model-facing text unless
	// skills.HintsEnabled is true (see message). It is printed at most once per
	// text throttle (see message) per session (see sessionEnv), or per agent on
	// this machine when there is no session id (see fallbackKey).
	text string

	// textKind tags the hint-shown event logged when text prints. With
	// skills.HintsEnabled on, message gives the skills list kind instead.
	textKind protos.AitoolsHintKind

	// sessionEnv names the env var carrying the agent's session id, which keys
	// the throttles of both lines. When it is empty or unset at runtime,
	// fallbackKey keys them instead.
	sessionEnv string

	// sessionKeyPrefix namespaces the session id in the throttle key, so one
	// agent's marker never silences another agent that reuses the same id.
	// Empty for Claude Code, whose key stays the raw session id so its
	// throttle is unchanged.
	sessionKeyPrefix string

	// fallbackKey keys the throttle when there is no session id, so the hint
	// prints at most once per text throttle on this machine. Empty means no
	// throttle without a session id (Claude Code).
	fallbackKey string

	// durableThrottle prints the text only once its throttle marker is
	// persisted (see claimHintMarker), so an unwritable cache can't turn the
	// hint into a line on every command. False for Claude Code, whose marker
	// write stays best-effort so its existing throttle behaviour is unchanged.
	durableThrottle bool

	// native returns a line the agent itself acts on (for Claude Code, its
	// plugin install prompt), or "" when none applies to this invocation. It is
	// throttled per session by nativeHintThrottle, only when it prints. Nil for
	// agents without one (all but Claude Code).
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
			agentName:  NameClaudeCode,
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

// isHintedAgent reports whether MaybeHint hints the agent with this
// useragent.AgentProvider() value. It defers to hintForAgent so both follow
// the same dispatch; pluginRecorded is only read when the hint's installed
// check runs, so none is needed here.
func isHintedAgent(provider string) bool {
	return hintForAgent(provider, nil) != nil
}

// nativeLine returns the agent-native line due for this invocation, or "".
func (h *agentHint) nativeLine(ctx context.Context) string {
	if h.native == nil {
		return ""
	}
	return h.native(ctx)
}

// MaybeHint nudges a supported agent toward Databricks AI tooling when it drives
// the CLI without it installed. It is a no-op when DATABRICKS_AGENT_FEATURES_DISABLED
// is set, and for human callers,
// unsupported agents, hint-exempt commands (isHintExemptCommand), the
// `databricks skills` commands when skills.HintsEnabled is true, the `auth
// token` and `auth docker token` credential helpers, a CLI process the CLI
// started itself, when there is no output stream, and once the tooling is
// present.
// The model-facing text is the skills list recommendation when
// skills.HintsEnabled is true, else the agent's install text (see message).
// The agent-native line (if any) and the model-facing text are throttled
// independently, per session or, for agents without a session id, per agent
// on this machine (fallbackKey). Both hint-shown events follow the text's
// clock (see nativeEventThrottle).
// pluginRecorded reads the CLI install record for agents whose plugin has no
// manifest reader here (see Agent.DatabricksTooling).
// Best-effort: it never fails the command. Besides the stderr lines, it logs a
// telemetry event for each hint line shown, which makes a command that resolved
// auth upload telemetry at exit.
//
// When the auto-install after cmd may give the agent its tools (see
// autoinstall.InstallsFor), the hint is withheld: MaybeHint returns a function
// to call once the command has ended, which prints the hint unless the tooling
// is installed by then. A failed command, a gate that turns out off or a
// failed install then still gets the hint. Otherwise it returns nil.
func MaybeHint(ctx context.Context, cmd *cobra.Command, pluginRecorded PluginRecordedFunc) (deferred func()) {
	if skills.AgentFeaturesDisabled(ctx) {
		return nil
	}
	hint := hintForAgent(useragent.AgentProvider(), pluginRecorded)
	if hint == nil {
		return nil
	}
	if isHintExemptCommand(cmd) {
		return nil
	}
	// Only with skill hints on: with them off, the skills commands keep the
	// install hint they had before the skills list hint existed.
	if skills.HintsEnabled(ctx) && isUnder(cmd, skillsCommandPath) {
		return nil
	}
	if IsCredentialHelperCommand(cmd) {
		return nil
	}
	if IsSelfExec(ctx) {
		return nil
	}
	if !cmdio.HasIO(ctx) {
		return nil
	}
	// Check the throttles before the filesystem probe, so a throttled session
	// skips the plugin/skills detection. A marker is written only once its
	// line is due, so an already-installed environment never burns one, and
	// neither does a native line withheld from this invocation.
	key := hint.throttleKey(ctx)
	native := hint.nativeLine(ctx)
	if native != "" && nativeHintThrottle.seen(ctx, key) {
		native = ""
	}
	textDue := !hint.message(ctx).throttle.seen(ctx, key)
	if native == "" && !textDue {
		return nil
	}
	// An agent that already has the tooling gets neither text, with skills hints
	// on too: which agents are hinted stays as it was.
	if hint.installed(ctx) {
		return nil
	}
	// The auto-install after this command prints its own notice. The install
	// text must not ask for an install that run already makes, so both texts
	// are withheld while it may install: one rule for both. No marker is
	// written until the hint prints.
	if autoinstall.InstallsFor != nil && autoinstall.InstallsFor(cmd, hint.agentName) {
		return func() {
			if hint.installed(ctx) {
				return
			}
			hint.show(ctx, key, native, textDue)
		}
	}
	hint.show(ctx, key, native, textDue)
	return nil
}

// show prints the lines MaybeHint found due, records their throttle markers
// and logs their hint-shown events.
func (h *agentHint) show(ctx context.Context, key, native string, textDue bool) {
	msg := h.message(ctx)
	// Only agents without a native line have a durable throttle, so a failed
	// claim leaves nothing to print.
	if textDue && !h.claim(ctx, msg.throttle, key) {
		return
	}
	if native != "" {
		cmdio.LogString(ctx, native)
		nativeHintThrottle.record(ctx, key)
		// A text print starts a new text window, in which this is the first
		// native print.
		if textDue || !nativeEventThrottle.seen(ctx, key) {
			logHintShown(ctx, h.nativeKind)
			nativeEventThrottle.set(ctx, key, true)
		}
	} else if textDue && h.native != nil {
		nativeEventThrottle.set(ctx, key, false)
	}
	if textDue {
		cmdio.LogString(ctx, msg.text)
		logHintShown(ctx, msg.kind)
	}
}

// hintMessage is the model-facing text of the hint, with the hint-shown kind
// its event logs and the throttle that gates it.
type hintMessage struct {
	text     string
	kind     protos.AitoolsHintKind
	throttle hintThrottle
}

// message returns the text, kind and throttle of the hint: the skills list
// ones when skills.HintsEnabled is true, else the agent's install text, its
// kind and the install throttle. They are picked together behind one guard, so
// the logged kind and the throttle always match the text that prints.
func (h *agentHint) message(ctx context.Context) hintMessage {
	if skills.HintsEnabled(ctx) {
		return hintMessage{text: skillsHintText, kind: protos.AitoolsHintKindSkillsListText, throttle: skillsTextHintThrottle}
	}
	return hintMessage{text: h.text, kind: h.textKind, throttle: textHintThrottle}
}

// claim records the text's throttle key and reports whether the text may
// print. A durable throttle allows it only once the marker has persisted.
func (h *agentHint) claim(ctx context.Context, throttle hintThrottle, key string) bool {
	if h.durableThrottle {
		return claimHintMarker(ctx, throttle, key)
	}
	throttle.record(ctx, key)
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
	if env.Get(ctx, claudeChildSessionEnv) == "" || InCI(ctx) {
		return ""
	}
	if cmdio.HasTerminal(ctx) {
		return ""
	}
	return claudeNativeHint
}

// InCI reports whether the CLI runs in CI: the generic CI variable set to
// anything but an explicit false (some providers set it to their own name), or
// any provider the SDK detects for its user agent (e.g. Azure Pipelines'
// TF_BUILD, which doesn't set CI). The false values are the ones env.GetBool
// accepts; GetBool itself can't be used because it also maps a provider name
// to false.
func InCI(ctx context.Context) bool {
	switch strings.ToLower(env.Get(ctx, ciEnv)) {
	case "", "0", "f", "false", "no", "off":
	default:
		return true
	}
	return useragent.CiCdProvider() != ""
}

// IsCredentialHelperCommand reports whether cmd is `databricks auth token` or
// `databricks auth docker token`.
func IsCredentialHelperCommand(cmd *cobra.Command) bool {
	if cmd.Name() != tokenCommandName {
		return false
	}
	parent := cmd.Parent()
	if parent != nil && parent.Name() == dockerCommandName {
		parent = parent.Parent()
	}
	return parent != nil && parent.Name() == authCommandName
}

// skillsCommandPath is the `databricks skills` command group. Its commands are
// hint-exempt only when skills.HintsEnabled is true (see MaybeHint): reading a
// skill is the task itself, so the skills list hint would only repeat it.
const skillsCommandPath = "skills"

// hintExemptCommands are the commands, with their subcommands, on which no
// tooling hint prints. Paths are relative to the root command. The aitools
// trees are exempt because telling the agent to run aitools while it already
// runs aitools makes no sense. The rest are plumbing or one-shot lookups (auth
// checks, "who am I", version, help, shell completion): the agent has no task
// to apply the skills to. The nudge eval measured only "who am I": 0 of 20
// runs installed the tooling.
var hintExemptCommands = []string{
	"aitools",
	"experimental aitools",
	"auth",
	"configure",
	"current-user me",
	"version",
	"feedback", // A one-shot report: the agent has no task to apply the skills to.
	"completion",
	"help",
	cobra.ShellCompRequestCmd,
}

// isHintExemptCommand reports whether cmd is one of hintExemptCommands or a
// subcommand of one.
func isHintExemptCommand(cmd *cobra.Command) bool {
	for _, exempt := range hintExemptCommands {
		if isUnder(cmd, exempt) {
			return true
		}
	}
	return false
}

// isUnder reports whether cmd is the command at path, relative to the root
// command, or one of its subcommands.
func isUnder(cmd *cobra.Command, path string) bool {
	full := cmd.Root().CommandPath() + " " + path
	cmdPath := cmd.CommandPath()
	return cmdPath == full || strings.HasPrefix(cmdPath, full+" ")
}

// claudeToolingInstalled reports whether Claude Code has Databricks AI tooling
// in its own locations (see Agent.DatabricksTooling): the databricks plugin, or
// databricks skills in Claude's skills dirs. Skills present only in the shared
// ~/.databricks/aitools/skills store don't count, because Claude Code doesn't
// load them from there. A plugin recorded with project or local scope counts
// only when the CLI's working directory is inside the recorded project: Claude
// Code loads it from the directory it started in, which the CLI can't see, so
// a session started in the project that runs `cd /tmp/x && databricks ...` is
// hinted. Any read/parse failure reports false, so an unreadable state errs
// toward hinting.
func claudeToolingInstalled(ctx context.Context) bool {
	claude := ByName(NameClaudeCode)
	if claude == nil {
		// claude-code is always in the registry; this guard keeps the hint
		// best-effort rather than panicking in a persistent pre-run hook.
		return false
	}
	// Claude Code's own manifest decides, so the CLI's install record is never
	// consulted for it.
	tooling := claude.DatabricksTooling(ctx, noRecordedPlugin)
	if tooling == ToolingPlugin {
		log.Debugf(ctx, "Databricks plugin detected for Claude Code; skipping install hint")
	}
	return tooling != ToolingNone
}

// throttleKey returns the cache key throttling the hint: the session id behind
// sessionKeyPrefix, else fallbackKey. "" means no throttle.
func (h *agentHint) throttleKey(ctx context.Context) string {
	if session := sessionID(ctx, h.sessionEnv); session != "" {
		return h.sessionKeyPrefix + session
	}
	return h.fallbackKey
}

// markers returns the file cache holding t's markers.
func (t hintThrottle) markers(ctx context.Context) *cache.Cache {
	return cache.NewCache(ctx, t.component, t.ttl, nil)
}

// seen reports whether key was marked true within the last t.ttl. With no key
// it returns false so the line prints per command.
func (t hintThrottle) seen(ctx context.Context, key string) bool {
	if key == "" {
		return false
	}
	marked, ok := cache.Get[bool](ctx, t.markers(ctx), key)
	return ok && marked
}

// record marks key, so later commands skip the line until the marker expires
// (t.ttl). No-op when the key is "".
func (t hintThrottle) record(ctx context.Context, key string) {
	t.set(ctx, key, true)
}

// set writes key's marker, restarting its ttl; seen reports it only when
// marked is true. No-op when the key is "".
func (t hintThrottle) set(ctx context.Context, key string, marked bool) {
	if key == "" {
		return
	}
	cache.Put(ctx, t.markers(ctx), key, marked)
}

// sessionID returns the value of sessionEnv, or "" when the agent has no
// session env var or it is unset.
func sessionID(ctx context.Context, sessionEnv string) string {
	if sessionEnv == "" {
		return ""
	}
	return env.Get(ctx, sessionEnv)
}

// IsSelfExec reports whether this CLI process was started by the CLI itself
// (see SelfExecEnv).
func IsSelfExec(ctx context.Context) bool {
	return env.Get(ctx, selfExecEnv) != ""
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
// auth: a hint shown on a command that never resolves auth is not uploaded.
func logHintShown(ctx context.Context, kind protos.AitoolsHintKind) {
	telemetry.Log(ctx, protos.DatabricksCliLog{
		AitoolsHintEvent: &protos.AitoolsHintEvent{HintKind: kind},
	})
}
