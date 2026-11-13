package agents

import (
	"context"
	"os"
	"path/filepath"
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
	// airSkillName is the agent skill for `databricks air`.
	airSkillName = "databricks-ai-runtime"

	// airHintText recommends the AI Runtime skill on the commands it helps with.
	// It names the skill and what it buys the agent, so a coding agent can tell
	// the hint is about the task at hand rather than generic setup, and offers a
	// --path preview so the agent can read the skill without installing anything.
	airHintText = "Tip: the " + airSkillName + " skill is not installed. It documents the air workload YAML, GPU types, secrets, and how to follow and debug runs. " +
		"Install it: databricks aitools install --skills-only --skills " + airSkillName +
		" (to read it first without installing: databricks aitools install --path <dir> --skills " + airSkillName + ")"

	// claudeAgentProvider is the useragent.AgentProvider() value for Claude Code.
	// The SDK maps the CLAUDECODE env var to this product name. The hint is
	// Claude-only for now (detection reads Claude's own on-disk state).
	// https://github.com/databricks/databricks-sdk-go/blob/main/useragent/agent.go
	claudeAgentProvider = "claude-code"

	// airCommandName is the top-level command whose runnable subcommands get the
	// hint. Bare `databricks air` and `--help` render help without running the
	// pre-run hook; the air help footer carries the same recommendation there.
	airCommandName = "air"

	// claudeSessionEnv is Claude Code's per-session identifier. It keys the
	// per-session throttle (at most one hint per hintSessionTTL per session);
	// when unset (older Claude) the hint degrades to per-command.
	claudeSessionEnv = "CLAUDE_CODE_SESSION_ID"

	// hintCacheComponent names the cache bucket holding per-session "already
	// hinted" markers.
	hintCacheComponent = "aitools-hint"
)

// hintSessionTTL is how long a session's "already hinted" marker lives. Within
// that window a session is hinted at most once; after it expires the hint may
// fire again for the same session.
const hintSessionTTL = time.Hour

// MaybeHint prints a one-line recommendation to install the AI Runtime skill
// when Claude Code runs a `databricks air` subcommand without the skill
// available. Other commands are never hinted: an agent running them has no AI
// Runtime task to apply the skill to, so the line is noise it learns to skip.
// It is also a no-op for human callers, other agents, when there is no output
// stream, and when the session was hinted within the last hintSessionTTL.
// Best-effort: it never fails the command. Besides the stderr line, it logs a
// telemetry event for each hint shown, which makes a command that resolved
// auth upload telemetry at exit.
func MaybeHint(ctx context.Context, cmd *cobra.Command) {
	if useragent.AgentProvider() != claudeAgentProvider {
		return
	}
	if !isAIRSubcommand(cmd) {
		return
	}
	if !cmdio.HasIO(ctx) {
		return
	}
	// Check the session throttle before the filesystem probe: once a session has
	// been hinted, later commands short-circuit here and skip the plugin/skills
	// detection. The throttle token is written only after a hint prints (below),
	// so an already-installed environment never burns it.
	if alreadyHintedThisSession(ctx) {
		return
	}
	if airSkillAvailable(ctx) {
		return
	}
	cmdio.LogString(ctx, airHintText)
	logHintShown(ctx, protos.AitoolsHintKindInstallText)
	recordSessionHint(ctx)
}

// isAIRSubcommand reports whether cmd sits under the top-level air command
// (`databricks air list`, `databricks air pools list`, ...), excluding air
// itself. Matching on the top-level ancestor keeps hidden aliases such as
// `databricks experimental air` out.
func isAIRSubcommand(cmd *cobra.Command) bool {
	for c := cmd; c.HasParent(); c = c.Parent() {
		if !c.Parent().HasParent() {
			return c != cmd && c.Name() == airCommandName
		}
	}
	return false
}

// airSkillAvailable reports whether Claude Code has the AI Runtime skill in the
// current working directory: the databricks plugin (which bundles every
// Databricks skill) installed for every project or for this one, or a
// databricks-ai-runtime skill dir in the shared canonical or legacy store or in
// the project store of cwd or a parent up to the repository root. Other
// Databricks skills do not count: they don't cover air. Any read/parse failure
// reports false, so an unreadable state errs toward hinting.
func airSkillAvailable(ctx context.Context) bool {
	cwd := ""
	if wd, err := os.Getwd(); err == nil {
		cwd = resolvePath(wd)
	}
	if claude := ByName(NameClaudeCode); claude != nil && claude.pluginInstalls != nil {
		if active, _ := claude.DatabricksPluginActiveIn(ctx, cwd); active {
			log.Debugf(ctx, "Databricks plugin active for Claude Code in %q; skipping AI Runtime skill hint", cwd)
			return true
		}
	}

	var dirs []string
	if homeDir, err := env.UserHomeDir(ctx); err == nil {
		dirs = append(dirs, filepath.Join(homeDir, CanonicalSkillsDir), filepath.Join(homeDir, LegacySkillsDir))
	}
	if cwd != "" {
		for _, dir := range projectDirs(cwd) {
			dirs = append(dirs, filepath.Join(dir, CanonicalSkillsDir))
		}
	}
	for _, dir := range dirs {
		// Stat follows the symlinks the installer may leave in place of copies.
		if info, err := os.Stat(filepath.Join(dir, airSkillName)); err == nil && info.IsDir() {
			return true
		}
	}
	return false
}

// alreadyHintedThisSession reports whether this Claude session was hinted
// within the last hintSessionTTL. With no session id it returns false so the
// hint fires per command.
func alreadyHintedThisSession(ctx context.Context) bool {
	session := env.Get(ctx, claudeSessionEnv)
	if session == "" {
		return false
	}
	c := cache.NewCache(ctx, hintCacheComponent, hintSessionTTL, nil)
	_, seen := cache.Get[bool](ctx, c, session)
	return seen
}

// recordSessionHint marks the current Claude session as hinted, so later
// commands in the same session stay quiet until the marker expires
// (hintSessionTTL). No-op when the session id is unset.
func recordSessionHint(ctx context.Context) {
	session := env.Get(ctx, claudeSessionEnv)
	if session == "" {
		return
	}
	c := cache.NewCache(ctx, hintCacheComponent, hintSessionTTL, nil)
	cache.Put(ctx, c, session, true)
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
