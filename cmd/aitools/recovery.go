package aitools

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/shellquote"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/versioncheck"
)

const (
	skillsReleasesURL = "https://github.com/databricks/databricks-agent-skills/releases"
	cliIssuesURL      = "https://github.com/databricks/cli/issues"
	// debugFlag is added to a re-run command whose output should explain a
	// failure the step cannot.
	debugFlag = "--debug"
)

// installInvocation is the subset of install flags a re-run must repeat: this
// run's own choices, so a copied command does the same thing and never widens
// to every agent or the global scope. Values are quoted for the user's shell
// when the command is rendered.
type installInvocation struct {
	path, scope, agents, skills          string
	skillsOnly, experimental, jsonOutput bool
}

// renderInstallCommand renders inv as a command line for the shell a user on
// goos pastes it into. It reports false when a value cannot be quoted so that
// the shell passes it on unchanged.
func renderInstallCommand(goos string, inv installInvocation) (string, bool) {
	args := []string{"databricks aitools install"}
	if inv.path != "" {
		path := inv.path
		if goos == "windows" {
			// Windows accepts forward slashes, and so the command pastes the
			// same into cmd.exe, PowerShell and Git Bash, which would drop
			// the backslashes. filepath.ToSlash follows the host OS, not goos.
			path = strings.ReplaceAll(path, `\`, "/")
		}
		path, ok := shellquote.Arg(goos, path)
		if !ok {
			return "", false
		}
		args = append(args, "--path="+path)
	}
	if inv.scope != "" {
		args = append(args, "--scope="+inv.scope)
	}
	if inv.agents != "" {
		// Quoted like --skills: PowerShell reads a bare comma as the array
		// operator and would split the list into two arguments.
		names, ok := shellquote.Arg(goos, inv.agents)
		if !ok {
			return "", false
		}
		args = append(args, "--agents="+names)
	}
	if inv.skillsOnly {
		args = append(args, "--skills-only")
	}
	if inv.skills != "" {
		// Normalized as install parses it, so "a, b" pastes as one value.
		skills, ok := shellquote.Arg(goos, strings.Join(splitAndTrim(inv.skills), ","))
		if !ok {
			return "", false
		}
		args = append(args, "--skills="+skills)
	}
	if inv.experimental {
		args = append(args, "--experimental")
	}
	if inv.jsonOutput {
		args = append(args, "--output json")
	}
	return strings.Join(args, " "), true
}

// commandText renders inv followed by extra flags, for printing after a colon.
// When a value cannot be printed so that it pastes literally, it returns false
// and a phrase asking for the user's own command with the flags in changed
// (those this command sets differently) and extra.
func commandText(goos string, inv installInvocation, changed []string, extra ...string) (string, bool) {
	cmd, ok := renderInstallCommand(goos, inv)
	if !ok {
		phrase := "your original install command"
		if flags := slices.Concat(changed, extra); len(flags) > 0 {
			phrase += " with " + strings.Join(flags, " ")
		}
		return phrase, false
	}
	return strings.Join(append([]string{cmd}, extra...), " "), true
}

// inSentence delimits a command printed inside a sentence. Single quotes are
// the convention for commands in CLI messages; a command that contains one (a
// value quoted for a POSIX shell) is delimited with backticks instead, so the
// reader can tell where it ends. A phrase that is not a command stays as is.
func inSentence(text string, isCommand bool) string {
	switch {
	case !isCommand:
		return text
	case strings.Contains(text, "'"):
		return "`" + text + "`"
	default:
		return "'" + text + "'"
	}
}

// rerunPhrase names the command that repeats inv plus extra flags, for use
// inside a sentence.
func rerunPhrase(goos string, inv installInvocation, extra ...string) string {
	return inSentence(commandText(goos, inv, nil, extra...))
}

// recoveryError appends a recovery step to an install failure. It unwraps to the
// original error so telemetry classification is unchanged.
type recoveryError struct {
	err      error
	recovery string
}

// Error appends the recovery step without its final period: an error string
// must not end with punctuation.
func (e *recoveryError) Error() string {
	return e.err.Error() + "\n" + strings.TrimSuffix(e.recovery, ".")
}
func (e *recoveryError) Unwrap() error { return e.err }

// displayError replaces the message of an error that is already explained by
// lines printed above it, keeping the cause for classification.
type displayError struct {
	msg string
	err error
}

func (e *displayError) Error() string { return e.msg }
func (e *displayError) Unwrap() error { return e.err }

// withRecovery attaches the recovery step for err. The install command returns
// its run error through it, which is nil when the run succeeded.
func withRecovery(ctx context.Context, inv installInvocation, err error) error {
	if err == nil {
		return nil
	}
	// A step attached closer to the failure (see skillsBatchError) wins.
	if _, ok := errors.AsType[*recoveryError](err); ok {
		return err
	}
	step := installRecovery(ctx, inv, err)
	if step == "" {
		return err
	}
	return &recoveryError{err: err, recovery: step}
}

// installRecovery returns the next step a user can take after a failed install.
// It is keyed on the category classifyInstallError reports, so the advice and
// the telemetry category cannot disagree. It returns "" when the error already
// says what to do.
func installRecovery(ctx context.Context, inv installInvocation, err error) string {
	if _, ok := errors.AsType[*agentErrors](err); ok {
		// Each agent's failure was printed with its own step.
		return ""
	}
	if errors.Is(err, installer.ErrNoSkillsResolved) {
		step := installer.NoSkillsResolvedMessage + " Re-run " + rerunPhrase(runtime.GOOS, inv, debugFlag) + " to see why each skill was skipped."
		if !inv.experimental {
			step += " To include experimental skills, add --experimental."
		}
		return step
	}
	// An empty picker selection has no category of its own: the funnel
	// reports it as CANCELLED.
	if errors.Is(err, errNoAgentsSelected) {
		return promptRecovery(inv)
	}
	rerun := rerunPhrase(runtime.GOOS, inv)
	// classifyInstallError reports RATE_LIMITED and SKILLS_FETCH_FAILED only for
	// a FetchError, and STATE_INVALID only for a StateError.
	ferr, _ := errors.AsType[*installer.FetchError](err)
	switch classifyInstallError(err) {
	case protos.AitoolsErrorCategoryCommandInterrupted:
		// The user stopped the command; nothing failed that needs a fix.
		return ""
	case protos.AitoolsErrorCategoryUserAborted, protos.AitoolsErrorCategoryPromptFailed:
		return promptRecovery(inv)
	case protos.AitoolsErrorCategoryNetwork, protos.AitoolsErrorCategoryTimeout:
		// A fetch that got no response; other network errors get the generic step.
		if ferr == nil {
			return genericRecovery(inv)
		}
		if ferr.Stage == installer.FetchStageRelease {
			return releaseLookupRecovery(ctx, rerun)
		}
		return unreachableRecovery(ctx, rerun)
	case protos.AitoolsErrorCategoryReleaseLookupFailed:
		return releaseLookupRecovery(ctx, rerun)
	case protos.AitoolsErrorCategoryRateLimited:
		if ferr.Stage == installer.FetchStageRelease {
			return releaseHost(ctx) + " rate-limited the lookup of the latest Databricks skills release. " + releaseLookupStep(rerun)
		}
		return skillsHost(ctx) + " rate-limited the download (HTTP " + strconv.Itoa(ferr.StatusCode) + "). Wait a few minutes and re-run " + rerun + "."
	case protos.AitoolsErrorCategoryManifestNotFound:
		if env.Get(ctx, "DATABRICKS_SKILLS_REF") != "" {
			return "No skills release exists at the version DATABRICKS_SKILLS_REF names. Unset it, or set it to a release tag from " + skillsReleasesURL + ", then re-run " + rerun + "."
		}
		return "The skills release this CLI selected is missing. Set DATABRICKS_SKILLS_REF to a release tag from " + skillsReleasesURL + " and re-run " + rerun + ", and report it at " + cliIssuesURL + "."
	case protos.AitoolsErrorCategoryManifestInvalid:
		host := skillsHost(ctx)
		// The manifest arrived and decoded, but lists a path outside the
		// skills directory or a skill in a form it cannot be installed in.
		if _, ok := errors.AsType[*installer.SkillError](err); ok || errors.Is(err, installer.ErrUnsafeManifestPath) {
			return "No skill files were written. Set DATABRICKS_SKILLS_REF to an earlier release tag from " + skillsReleasesURL + " and re-run " + rerun + ", and report the manifest from " + host + " at " + cliIssuesURL + "."
		}
		return host + " answered with something that is not a skills manifest (a proxy or captive portal page, or a broken mirror). Check what this machine gets from " + host + ", then re-run " + rerun + "."
	case protos.AitoolsErrorCategorySkillsFetchFailed:
		return downloadRecovery(ctx, inv, ferr)
	case protos.AitoolsErrorCategoryStateInvalid:
		serr, _ := errors.AsType[*installer.StateError](err)
		return stateRecovery(inv, serr)
	case protos.AitoolsErrorCategorySkillNotFound:
		return "Run 'databricks aitools list' to see the available skills."
	case protos.AitoolsErrorCategoryVersionIncompatible:
		if _, cmd := versioncheck.DetectInstallMethod(ctx); cmd != "" {
			return "Upgrade the Databricks CLI with '" + cmd + "', then re-run the install."
		}
		return "Upgrade the Databricks CLI (see " + root.InstallDocsURL + "), then re-run the install."
	case protos.AitoolsErrorCategoryExperimentalSkill:
		// The message already names --experimental.
		return ""
	case protos.AitoolsErrorCategoryPermissionDenied:
		return permissionRecovery(inv, err)
	case protos.AitoolsErrorCategoryFilesystem:
		return filesystemRecovery(inv, err)
	default:
		// A refusal is UNCATEGORIZED and its message names the fix. Checked
		// here, not before the switch, so a failure joined to a refusal still
		// gets the step for its own category.
		if _, ok := errors.AsType[*installer.RefusedError](err); ok {
			return ""
		}
		return genericRecovery(inv)
	}
}

// genericRecovery is the step for a failure no other step explains: re-run
// with debug output, and report it if it persists.
func genericRecovery(inv installInvocation) string {
	return "Re-run " + rerunPhrase(runtime.GOOS, inv, debugFlag) + " to see details. If it keeps failing, report it at " + cliIssuesURL + "."
}

// unreachableRecovery is the step for a skills download that got no response,
// or whose response broke off.
func unreachableRecovery(ctx context.Context, rerun string) string {
	return "Could not download Databricks skills from " + skillsHost(ctx) + ". Check that this machine can reach it (set HTTPS_PROXY if you use a proxy), then re-run " + rerun + "."
}

// promptRecovery is the step for a prompt that was aborted or could not be
// shown: name every choice by flag, so the re-run prompts for nothing.
func promptRecovery(inv installInvocation) string {
	if inv.scope == "" {
		inv.scope = installer.ScopeGlobal
	}
	var extra []string
	if inv.agents == "" {
		extra = append(extra, "--agents=<name>")
	}
	cmd, _ := commandText(runtime.GOOS, inv, []string{"--scope=" + inv.scope}, extra...)
	return "To install without prompts, name the scope and agents, for example: " + cmd
}

// releaseLookupRecovery is the step for a lookup of the latest skills release
// that failed for any reason but a rate limit.
func releaseLookupRecovery(ctx context.Context, rerun string) string {
	return "Could not look up the latest Databricks skills release on " + releaseHost(ctx) + ". " + releaseLookupStep(rerun)
}

// releaseLookupStep is what to do after a failed lookup of the latest skills
// release: wait and retry, or pin a release so no lookup is needed.
func releaseLookupStep(rerun string) string {
	return "Wait a few minutes and re-run " + rerun + ", or skip the lookup by setting DATABRICKS_SKILLS_REF to a release tag from " + skillsReleasesURL + "."
}

// releaseHost is the host the latest skills release is looked up on.
func releaseHost(ctx context.Context) string {
	return hostOf(versioncheck.GitHubAPIURL(ctx))
}

// skillsHost is the host the skills manifest and files are downloaded from.
func skillsHost(ctx context.Context) string {
	return hostOf(installer.GetSkillsBaseURL(ctx))
}

// hostOf returns the host of rawURL. An environment override without a scheme
// has no host to name, so it is returned as it was set: that is what the CLI
// tried to fetch.
func hostOf(rawURL string) string {
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		return u.Host
	}
	return rawURL
}

// downloadRecovery is the step for a manifest or skill file download that got
// a response. A status below 400 means the body could not be read (a dropped
// connection), which is a network failure too.
func downloadRecovery(ctx context.Context, inv installInvocation, ferr *installer.FetchError) string {
	host := skillsHost(ctx)
	rerun := rerunPhrase(runtime.GOOS, inv)
	status := strconv.Itoa(ferr.StatusCode)
	switch {
	case ferr.StatusCode >= 500:
		return host + " returned HTTP " + status + ", a server-side error. Wait a few minutes and re-run " + rerun + "."
	case ferr.StatusCode == http.StatusForbidden || ferr.StatusCode == http.StatusProxyAuthRequired:
		return host + " refused the download (HTTP " + status + "). A proxy or access policy may block it; ask your network admin to allow " + host + ", then re-run " + rerun + "."
	case ferr.StatusCode >= 400:
		return host + " returned HTTP " + status + ". Re-run " + rerunPhrase(runtime.GOOS, inv, debugFlag) + " to see details, and report it at " + cliIssuesURL + "."
	default:
		return unreachableRecovery(ctx, rerun)
	}
}

// stateRecovery is the step for a state file that cannot be parsed. Without
// state, an install of named skills in the global scope is refused while other
// skills are on disk (they would stay untracked), so the global re-run installs
// all skills; a project-scope install of named skills is not refused, so it
// keeps --skills, and the step adds that the project's other skills need a
// re-run too, or the new state file leaves them untracked. The state file
// also records the plugins and skills of other
// agents in the scope, which a re-run limited to some agents, or to skill
// files, does not record again.
func stateRecovery(inv installInvocation, serr *installer.StateError) string {
	step := fmt.Sprintf("Move the install state file %s aside (for example, rename it to %s.bak), then re-run ", filepath.ToSlash(serr.Path), filepath.Base(serr.Path))
	named := inv.skills != "" && inv.scope == installer.ScopeGlobal
	if named {
		inv.skills = ""
	}
	step += rerunPhrase(runtime.GOOS, inv)
	if named {
		step += " to reinstall all skills (without the state file, an install of named skills in the global scope is refused)"
	}
	step += "."
	if inv.skills != "" {
		all := inv
		all.skills = ""
		step += " The new state file records only the named skills, so the project's other skills stay untracked by update and uninstall; reinstall them too with " + rerunPhrase(runtime.GOOS, all) + "."
	}
	if inv.agents != "" || inv.skillsOnly {
		step += " The state file also tracked the other agents set up in this scope; re-run their install too, for example " + rerunPhrase(runtime.GOOS, installInvocation{scope: inv.scope}) + "."
	}
	return step
}

// failedPath returns the path an OS error names, with forward slashes; a
// rename names its destination. It returns "" when the error names no path.
func failedPath(err error) string {
	if perr, ok := errors.AsType[*fs.PathError](err); ok {
		return filepath.ToSlash(perr.Path)
	}
	if lerr, ok := errors.AsType[*os.LinkError](err); ok {
		return filepath.ToSlash(lerr.New)
	}
	return ""
}

// permissionRecovery is the step for a file the install may not write.
func permissionRecovery(inv installInvocation, err error) string {
	rerun := rerunPhrase(runtime.GOOS, inv)
	path := failedPath(err)
	if path == "" {
		return "Check that you own the directories the install writes to and can write to them (an earlier run with sudo can leave them owned by root), then re-run " + rerun + "."
	}
	return fmt.Sprintf("Check that you own %s and can write to it (an earlier run with sudo can leave it owned by root), then re-run %s.", path, rerun)
}

// filesystemRecovery is the step for any other failure to write the install's
// files, such as a full disk or a read-only filesystem: neither is a CLI bug.
func filesystemRecovery(inv installInvocation, err error) string {
	rerun := rerunPhrase(runtime.GOOS, inv)
	path := failedPath(err)
	if path == "" {
		return "Check that the directories the install writes to are writable and that the disk has free space, then re-run " + rerun + "."
	}
	return fmt.Sprintf("Check that %s is on a writable filesystem with free space, then re-run %s.", path, rerun)
}

// agentCommandText renders the command that repeats this run for one agent,
// followed by extra flags. The scope is always spelled out: without it a
// non-interactive run defaults to global, which is not what a project-scoped
// run asked for.
func agentCommandText(inv installInvocation, it agentPlanItem, extra ...string) (string, bool) {
	changed := []string{"--scope=" + inv.scope, "--agents=" + it.agent.Name}
	inv.agents = it.agent.Name
	return commandText(runtime.GOOS, inv, changed, extra...)
}

// installCommand is agentCommandText for printing after a colon.
func installCommand(inv installInvocation, it agentPlanItem, extra ...string) string {
	cmd, _ := agentCommandText(inv, it, extra...)
	return cmd
}

// agentRerun is agentCommandText delimited for use inside a sentence.
func agentRerun(inv installInvocation, it agentPlanItem, extra ...string) string {
	return inSentence(agentCommandText(inv, it, extra...))
}

// skillsOnlyCommand is the raw-skills install for one agent, the escape hatch
// when its plugin cannot be installed.
func skillsOnlyCommand(inv installInvocation, it agentPlanItem) string {
	inv.skillsOnly = true
	return installCommand(inv, it)
}

// skillsAgentRerun is rerun for one agent that gets skill files. An agent
// whose CLI is missing gets --skills-only: named via --agents, it would get
// the plugin instead.
func skillsAgentRerun(inv installInvocation, it agentPlanItem) installInvocation {
	inv.agents = it.agent.Name
	inv.skillsOnly = inv.skillsOnly || it.cliMissing
	return inv
}

// pluginFailureText returns the one-line reason and the recovery step for a
// plugin agent that could not be set up. The skill files alternative starts
// with removing the plugin when the agent already has one, since --skills-only
// skips such an agent.
func pluginFailureText(ctx context.Context, inv installInvocation, it agentPlanItem, err error) (reason, recovery string) {
	a := it.agent
	skillFiles := skillsOnlyCommand(inv, it)
	if pluginAlreadyInstalledFn(ctx, a, inv.scope) {
		skillFiles = pluginRemovalStep(ctx, a, inv.scope) + ", then run: " + skillFiles
	}
	blocked, ok := errors.AsType[*installer.BlockedError](err)
	if !ok {
		return err.Error(), "Re-run " + agentRerun(inv, it, debugFlag) + " to see details, or install skill files instead: " + skillFiles
	}
	switch {
	case blocked.TimedOut:
		return fmt.Sprintf("'%s plugin' %s", a.Binary, blocked.Detail),
			"Re-run " + agentRerun(inv, it) + " (an agent CLI's first start can be slow), or install skill files instead: " + skillFiles
	case blocked.Reason == installer.ReasonCLINotOnPath:
		if a.HasBinary(ctx) {
			// The CLI is on PATH but has no working plugin command, e.g. a
			// version from before plugins.
			return fmt.Sprintf("'%s plugin' is not available (%s)", a.Binary, blocked.Detail),
				fmt.Sprintf("Update the %s CLI to a version with the '%s plugin' command, then re-run %s. Or install skill files instead: %s", a.DisplayName, a.Binary, agentRerun(inv, it), skillFiles)
		}
		return cliMissingReason(it),
			fmt.Sprintf("Install or update the %s CLI so '%s' is on PATH, then re-run %s. Or install skill files instead: %s", a.DisplayName, a.Binary, agentRerun(inv, it), skillFiles)
	case blocked.Reason == installer.ReasonShimExitedWithError:
		// Detail carries the captured stderr and the commands that show the
		// exit code; a plain re-run of the help may print only the help.
		return fmt.Sprintf("'%s plugin --help' %s", a.Binary, blocked.Detail),
			fmt.Sprintf("Once it exits with 0, re-run %s. Or install skill files instead: %s", agentRerun(inv, it), skillFiles)
	case blocked.Reason == installer.ReasonInstallFailed:
		manual := installer.ManualPluginInstallCommands(a, it.scope)
		return fmt.Sprintf("'%s plugin' failed: %s", a.Binary, blocked.Detail),
			"Run '" + strings.Join(manual, "', then '") + "' to see the full error. Once they succeed, re-run " + agentRerun(inv, it) + " so the CLI records the plugin. Or install skill files instead: " + skillFiles
	default:
		return err.Error(), "Install skill files instead: " + skillFiles
	}
}

// skipRecovery returns the recovery step for an agent the plan skipped. An
// agent skipped under --skills-only because it has the plugin gets the steps
// that replace the plugin with skill files. Every other skip is a project-scoped
// install the agent cannot take: an agent that reads project skill files was
// skipped only because its plugin is user-only (see mapAgentScope), so the step
// is the --skills-only install in the same scope. For any other agent the step
// is the command that sets up just this agent in the global scope.
func skipRecovery(inv installInvocation, it agentPlanItem) string {
	if it.pluginRemoval != "" {
		return "To use skill files instead, " + it.pluginRemoval + ", then run: " + skillsOnlyCommand(inv, it)
	}
	if it.agent.SupportsProjectSkills {
		return "To install skill files for " + it.agent.DisplayName + " in the project scope instead, run: " + skillsOnlyCommand(inv, it)
	}
	inv.scope = installer.ScopeGlobal
	return "To set up " + it.agent.DisplayName + " in the global scope, run: " + installCommand(inv, it)
}

// pluginRemovalStep is how the user removes the databricks plugin an agent
// already has. 'databricks aitools uninstall' removes only a plugin recorded in
// the CLI's install state, runs the agent's CLI to do it, and clears the
// record. For Claude Code the CLI reads Claude Code's own manifest, so each
// databricks plugin there that the record does not cover (one added in Claude
// Code itself, from any marketplace) gets Claude Code's own command, which
// also needs its CLI on PATH. A managed plugin comes from the organization's
// managed settings, which neither CLI can change, so an administrator has to
// remove it. An agent without a manifest reader (Codex, Copilot) has each
// databricks plugin its own config records (see configPluginMarketplaces) and
// the one the record names (see pluginAlreadyInstalled). A plugin of the
// config that the record does not cover gets the agent's own command; the
// record's plugin, or the record alone, has to be removed through 'databricks
// aitools uninstall', which also clears the record. Both need the agent's CLI
// on PATH.
func pluginRemovalStep(ctx context.Context, a *agents.Agent, scope string) string {
	uninstall := fmt.Sprintf("databricks aitools uninstall --scope=%s --agents=%s", scope, a.Name)
	if !a.HasPluginManifestReader() {
		rec, recorded := installer.RecordedPlugin(ctx, scope, a.Name)
		var own []string
		for _, marketplace := range configPluginMarketplaces(ctx, a, scope) {
			if recorded && marketplace == rec.Marketplace {
				continue
			}
			own = append(own, installer.PluginUninstallCommand(a, marketplace, ""))
		}
		if len(own) > 0 && !recorded {
			commands := strings.Join(own, "', then '")
			if !a.HasBinary(ctx) {
				return fmt.Sprintf("put '%s' on PATH and remove the plugin with '%s' first (this CLI did not install it)", a.Binary, commands)
			}
			return fmt.Sprintf("remove the plugin first with '%s' (this CLI did not install it)", commands)
		}
		commands := strings.Join(append([]string{uninstall}, own...), "', then '")
		if !a.HasBinary(ctx) {
			clears := "that"
			if len(own) > 0 {
				clears = "'" + uninstall + "'"
			}
			return fmt.Sprintf("put '%s' on PATH and remove the plugin with '%s' first (only %s clears this CLI's record of it)", a.Binary, commands, clears)
		}
		return fmt.Sprintf("remove the plugin first with '%s'", commands)
	}
	installs, err := manifestPluginInstalls(ctx, a, scope)
	isManaged := func(install agents.PluginInstall) bool { return install.Scope == agents.ClaudeScopeManaged }
	managed := slices.ContainsFunc(installs, isManaged)
	installs = slices.DeleteFunc(installs, isManaged)
	var step string
	switch {
	case managed && len(installs) == 0:
		return fmt.Sprintf("ask an administrator to remove the databricks plugin first (your organization manages it for %s, so you cannot remove it)", a.DisplayName)
	case !a.HasBinary(ctx):
		step = fmt.Sprintf("remove the databricks plugin in %s first ('%s' is not on PATH, so this CLI cannot remove it)", a.DisplayName, a.Binary)
	case err != nil:
		// The skip came from the CLI's record (see pluginAlreadyInstalled).
		return fmt.Sprintf("remove the plugin first with '%s'", uninstall)
	default:
		rec, recorded := installer.RecordedPlugin(ctx, scope, a.Name)
		var steps []string
		usesUninstall := false
		for _, install := range installs {
			if recorded && install.Marketplace == rec.Marketplace && install.Scope == rec.Scope {
				if !usesUninstall {
					steps = append(steps, uninstall)
					usesUninstall = true
				}
				continue
			}
			steps = append(steps, installer.PluginUninstallCommand(a, install.Marketplace, install.Scope))
		}
		step = "remove the plugin first with '" + strings.Join(steps, "', then '") + "'"
		if !usesUninstall {
			step += " (this CLI did not install it)"
		}
	}
	if managed {
		step += ", and ask an administrator to remove the one your organization manages"
	}
	return step
}

// cliMissingReason is why an agent whose CLI is not on PATH does not get the
// plugin.
func cliMissingReason(it agentPlanItem) string {
	return fmt.Sprintf("'%s' is not on PATH", it.agent.Binary)
}

// switchToPluginStep is how an agent whose CLI is missing gets the plugin later.
func switchToPluginStep(inv installInvocation, it agentPlanItem) string {
	return fmt.Sprintf("install the %s CLI so '%s' is on PATH and run %s", it.agent.DisplayName, it.agent.Binary, agentRerun(inv, it))
}

// skillsFallbackText is the notice printed when an agent gets skill files
// instead of the plugin because its CLI is missing.
func skillsFallbackText(inv installInvocation, it agentPlanItem) string {
	return fmt.Sprintf("Skipped the plugin for %s: %s. Installing skill files instead; to switch to the plugin, %s.",
		it.agent.DisplayName, cliMissingReason(it), switchToPluginStep(inv, it))
}

// projectPluginWarning is printed when a skills fallback runs in a directory
// where the agent already loads a plugin that does not cover the install scope.
func projectPluginWarning(scope string, it agentPlanItem) string {
	if scope == installer.ScopeProject {
		return it.agent.DisplayName + " has a personal databricks plugin, so you will see both the plugin and this project's skill files here."
	}
	return it.agent.DisplayName + " has the databricks plugin for this project, so this project will see both the plugin and the global skill files."
}

// pluginKeptText is the notice for an agent whose CLI is missing but which
// already has the plugin, so no skill files are added next to it.
func pluginKeptText(inv installInvocation, it agentPlanItem) string {
	return fmt.Sprintf("%s already has the databricks plugin; it was not refreshed because %s. To refresh it, %s.",
		it.agent.DisplayName, cliMissingReason(it), switchToPluginStep(inv, it))
}
