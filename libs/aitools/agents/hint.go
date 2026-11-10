package agents

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/cache"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

const (
	// hintText recommends installing the Databricks AI tooling. The wording
	// is deliberately chosen to be effective at helping agents; change it only
	// after thorough testing.
	hintText = "Databricks skills are not installed. To work with Databricks reliably, first run: databricks aitools install"

	// claudeAgentProvider is the useragent.AgentProvider() value for Claude Code.
	// The SDK maps the CLAUDECODE env var to this product name. The hint is
	// Claude-only for now (detection reads Claude's own on-disk state).
	// https://github.com/databricks/databricks-sdk-go/blob/main/useragent/agent.go
	claudeAgentProvider = "claude-code"

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

// MaybeHint prints a one-line recommendation to install the Databricks AI
// tooling when Claude Code drives the CLI without it installed. It is a no-op
// for human callers, other agents, hint-exempt commands (IsHintExemptCommand),
// when there is no output stream, once the tooling is present, and when the
// session was hinted within the last hintSessionTTL. Best-effort: it writes
// only to stderr and never fails the command.
func MaybeHint(ctx context.Context, cmd *cobra.Command) {
	if useragent.AgentProvider() != claudeAgentProvider {
		return
	}
	if IsHintExemptCommand(cmd) {
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
	if claudeToolingInstalled(ctx) {
		return
	}
	cmdio.LogString(ctx, hintText)
	recordSessionHint(ctx)
}

// hintExemptCommands are the commands, with their subcommands, on which no
// tooling hint prints. Paths are relative to the root command. The aitools
// trees are exempt because telling the agent to run aitools while it already
// runs aitools makes no sense. The rest are plumbing or one-shot lookups (auth
// checks, "who am I", help, shell completion): the agent has no task to apply
// the skills to, and the nudge eval showed such calls do not convert.
var hintExemptCommands = []string{
	"aitools",
	"experimental aitools",
	"auth",
	"configure",
	"current-user me",
	"completion",
	"help",
	cobra.ShellCompRequestCmd,
	cobra.ShellCompNoDescRequestCmd,
}

// IsHintExemptCommand reports whether cmd is one of hintExemptCommands or a
// subcommand of one.
func IsHintExemptCommand(cmd *cobra.Command) bool {
	path := commandPath(cmd)
	for _, exempt := range hintExemptCommands {
		if path == exempt || strings.HasPrefix(path, exempt+" ") {
			return true
		}
	}
	return false
}

// commandPath returns cmd's path below the root command, e.g. "current-user me",
// so hintExemptCommands holds whatever the root is named.
func commandPath(cmd *cobra.Command) string {
	var names []string
	for c := cmd; c.HasParent(); c = c.Parent() {
		names = append(names, c.Name())
	}
	slices.Reverse(names)
	return strings.Join(names, " ")
}

// claudeToolingInstalled reports whether Databricks AI tooling is present for
// Claude Code. It checks two signals: the databricks plugin in Claude's own
// plugin manifest (written both by `databricks aitools install` and by a direct
// `claude plugin install`), and databricks skill files in the shared canonical
// skills dir (written by `aitools install --skills-only`). Any read/parse
// failure reports false, so an unreadable state errs toward hinting.
// (*Agent).DatabricksTooling, which feeds the User-Agent, differs on purpose:
// it ignores the shared store and out-of-project records, and counts
// .claude/skills dirs, to report what the calling agent loads.
func claudeToolingInstalled(ctx context.Context) bool {
	claude := ByName(NameClaudeCode)
	if claude == nil {
		// claude-code is always in the registry; this guard keeps the hint
		// best-effort rather than panicking in a persistent pre-run hook.
		return false
	}
	if version, ok := claude.DatabricksPluginVersion(ctx); ok {
		log.Debugf(ctx, "Databricks plugin detected for Claude Code (version %q); skipping install hint", version)
		return true
	}

	// The canonical dir always holds a real databricks* subdir after a skills
	// install, regardless of whether the agent copy is a symlink or a copy. The
	// legacy dir is checked too so users of an older install aren't hinted.
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return false
	}
	return HasDatabricksSkillsIn(filepath.Join(home, CanonicalSkillsDir)) ||
		HasDatabricksSkillsIn(filepath.Join(home, LegacySkillsDir))
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
