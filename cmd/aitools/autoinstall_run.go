package aitools

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"os"
	"os/signal"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/versioncheck"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

const (
	// gateTimeout bounds the gate lookup, one compatibility manifest fetch.
	// A lookup that runs out reads as off: every machine pays it once per
	// version while the gate is off, so it is kept short.
	gateTimeout = 3 * time.Second

	// installTimeout bounds the install or update (its downloads and the
	// agent CLIs it runs). One that runs out of time counts as a failed
	// attempt and is retried on a later run.
	installTimeout = 10 * time.Second

	// promptTimeout bounds the wait for an answer at the install prompt.
	promptTimeout = time.Minute
)

// autoInstallDisable is the opt-out command named in the notice.
const autoInstallDisable = "databricks aitools auto-install disable"

// AutoInstall installs Databricks AI tools for the coding agents on this
// machine, or updates an existing installation, on the first run of a new CLI
// version. The root command runs it after a command succeeded. It never fails
// the command and never writes to stdout. On stderr it prints the install
// prompt (in a terminal) and one line on what it did; why it skipped or failed
// goes only to the debug log.
func AutoInstall(cmd *cobra.Command) {
	ctx := cmd.Context()
	defer func() {
		if r := recover(); r != nil {
			log.Debugf(ctx, "auto-install: panic: %v", r)
		}
	}()
	autoInstall(ctx, commandPath(cmd))
}

// commandPath returns the names of cmd and its parents below the root.
func commandPath(cmd *cobra.Command) []string {
	var path []string
	for c := cmd; c.HasParent(); c = c.Parent() {
		path = append([]string{c.Name()}, path...)
	}
	return path
}

func autoInstall(ctx context.Context, path []string) {
	version := autoinstall.Version(ctx)
	in := autoinstall.Inputs{
		Command:       path,
		CacheDisabled: autoinstall.CacheDisabled(ctx),
		Version:       version,
		CI:            versioncheck.InCI(ctx),
		// A "no" earlier in this process counts even if storing it failed.
		EnvOptOut: autoinstall.EnvOptedOut(ctx) || autoinstall.Declined(),
	}
	// Commands and builds that never auto-install skip before the state is
	// read, and without a log line: they are most runs.
	if reason := autoinstall.Decide(in); reason == autoinstall.ReasonSkippedCommand || reason == autoinstall.ReasonDevelopmentBuild {
		return
	}
	var err error
	in.State, err = autoinstall.Load(ctx)
	if err != nil {
		log.Debugf(ctx, "auto-install: skipped: %v", err)
		return
	}
	reason := autoinstall.Decide(in)
	if !reason.Due() {
		log.Debugf(ctx, "auto-install: skipped: %s", reason)
		return
	}
	// A run that would ask before installing but cannot skips before the lock
	// and the gate lookup: it would skip after them too, and it repeats on
	// every such run until a run in a terminal asks.
	if reason == autoinstall.ReasonGateOff && in.State.LastVersion != version && autoInstallAsks(ctx) && !cmdio.IsLinePromptSupported(ctx) {
		if tools, err := planTools(quietContext(ctx)); err == nil && len(tools.newAgents()) > 0 {
			skipUnaskable(ctx, in.State.ID, autoinstall.NewID())
			return
		}
	}

	unlock, ok := autoinstall.TryLock(ctx)
	if !ok {
		log.Debugf(ctx, "auto-install: skipped: another CLI process is running it")
		return
	}
	defer unlock()
	// Another process may have finished a run since the state was read.
	in.State, err = autoinstall.Load(ctx)
	if err != nil {
		log.Debugf(ctx, "auto-install: skipped: %v", err)
		return
	}
	id := in.State.ID
	if id == "" {
		id = autoinstall.NewID()
	}
	in.Bucket = autoinstall.Bucket(id)

	// The gate costs a manifest lookup, so it is resolved only when it is
	// what decides the run.
	reason = autoinstall.Decide(in)
	if reason == autoinstall.ReasonGateOff {
		gateCtx, cancel := context.WithTimeout(ctx, gateTimeout)
		in.GateOn = autoinstall.GateOn(gateCtx, version)
		cancel()
		reason = autoinstall.Decide(in)
	}
	run := autoInstallRun{
		version:     version,
		id:          id,
		arm:         holdoutArm(in.Bucket),
		interactive: autoInstallAsks(ctx),
	}
	if !reason.Due() {
		// Another process settled the version while this one waited.
		log.Debugf(ctx, "auto-install: skipped: %s", reason)
		return
	}
	if reason == autoinstall.ReasonGateOff {
		// The feature is not launched, so the skip is not reported.
		log.Debugf(ctx, "auto-install: skipped: %s", reason)
		if err := autoinstall.Record(ctx, version, 0, id); err != nil {
			log.Debugf(ctx, "auto-install: could not save the state: %v", err)
		}
		return
	}
	if outcome, ok := skipOutcomes[reason]; ok {
		run.settle(ctx, outcome)
		return
	}
	run.attempt(ctx, in.State)
}

// attempt installs or updates the tools and reports the outcome. prior is the
// state the run was decided on.
func (r autoInstallRun) attempt(ctx context.Context, prior autoinstall.State) {
	attempt := 1
	if prior.LastVersion == r.version {
		attempt = prior.Attempts + 1
	}
	qctx := quietContext(ctx)
	tools, err := planTools(qctx)
	// A terminal user is asked before the first install of a version, once:
	// a retry was answered "yes" (or ran non-interactively), so it is not
	// asked again.
	if r.interactive && attempt == 1 {
		switch {
		case err != nil:
			// A retry would install without asking, so this version settles.
			log.Debugf(ctx, "auto-install: failed before asking: %v", err)
			r.settleFailed(ctx, err)
			return
		case len(tools.newAgents()) > 0:
			// autoInstall skips this case before the lock; the state it
			// decided on may have changed since.
			if !cmdio.IsLinePromptSupported(ctx) {
				skipUnaskable(ctx, prior.ID, r.id)
				return
			}
			if !r.ask(ctx, tools.newAgents()) {
				return
			}
		}
	}
	// Count the attempt before making it: an attempt that never finishes (the
	// process is killed) still counts, and a state that cannot be saved stops
	// auto-install instead of retrying it on every run.
	if recErr := autoinstall.Record(ctx, r.version, attempt, r.id); recErr != nil {
		log.Debugf(ctx, "auto-install: skipped: could not save the state: %v", recErr)
		return
	}
	var result installResult
	if err == nil {
		runCtx, cancel := context.WithTimeout(qctx, installTimeout)
		result, err = tools.install(runCtx)
		cancel()
	}
	if _, refused := errors.AsType[*installer.RefusedError](err); refused {
		// The installed skills are newer than the release this CLI would
		// install (the downgrade guard). They are installed, and retrying
		// changes nothing until a newer release, which a later version gets.
		log.Debugf(ctx, "auto-install: kept the installed tools: %v", err)
		err = nil
	}
	switch {
	case err != nil:
		outcome := protos.AitoolsAutoInstallOutcomeFailed
		if attempt >= autoinstall.MaxAttempts {
			outcome = protos.AitoolsAutoInstallOutcomeRetriesExhausted
		}
		log.Debugf(ctx, "auto-install: attempt %d of %d failed: %v", attempt, autoinstall.MaxAttempts, err)
		r.report(ctx, outcome, classifyInstallError(err))
	case result.noAgents:
		r.settle(ctx, protos.AitoolsAutoInstallOutcomeSkippedNoAgents)
	default:
		r.settle(ctx, protos.AitoolsAutoInstallOutcomeInstalled)
		if len(result.agents) > 0 {
			verb := "Installed"
			if result.updated {
				verb = "Updated"
			}
			cmdio.LogString(ctx, fmt.Sprintf("%s Databricks AI tools for %s. Disable: %s", verb, strings.Join(result.agents, ", "), autoInstallDisable))
		}
	}
}

// autoInstallAsks reports whether a person may be at the terminal, so
// auto-install asks before installing: stdin, stdout or stderr is a terminal
// and no coding agent is detected (see installPromptsSupported). It asks only
// when the question can be seen and answered (cmdio.IsLinePromptSupported)
// and skips otherwise; the install without asking is left to runs with no
// terminal at all. The question is a plain-text line, so NO_COLOR, TERM=dumb
// and Git Bash, which turn off the TUI prompts of install, do not skip it.
func autoInstallAsks(ctx context.Context) bool {
	return cmdio.HasTerminal(ctx) && useragent.AgentProvider() == ""
}

// skipUnaskable skips a run that would ask before installing but cannot: the
// question could not be seen or answered safely, and installing without it
// would skip the consent. The version is not settled, so a later run in a
// terminal asks. A machine with no stored ID (stored) gets id, so that every
// such run does not draw a new holdout bucket.
func skipUnaskable(ctx context.Context, stored, id string) {
	log.Debugf(ctx, "auto-install: skipped: cannot ask (stdin, stdout or stderr is not a terminal, the CLI runs in the background, or stdin is still being read)")
	if stored != "" {
		return
	}
	if err := autoinstall.SaveID(ctx, id); err != nil {
		log.Debugf(ctx, "auto-install: could not save the state: %v", err)
	}
}

// ask asks whether to install for the agents names and reports whether to go
// ahead. The version is settled before asking, so the question is asked once
// per version even when it is never answered: the process is killed, stdin
// is closed, Ctrl+C is pressed, nobody answers within promptTimeout (an
// unrecognized agent in a pseudo-terminal), or the answer is neither yes nor
// no. Only "no" stores the opt-out. What was typed before the question, such
// as the next command typed while this one ran, is discarded unread.
func (r autoInstallRun) ask(ctx context.Context, names []string) bool {
	if err := autoinstall.Record(ctx, r.version, 0, r.id); err != nil {
		log.Debugf(ctx, "auto-install: skipped: could not save the state: %v", err)
		return false
	}
	if err := cmdio.DiscardInput(ctx); err != nil {
		log.Debugf(ctx, "auto-install: could not discard the pending input: %v", err)
	}
	actx, cancel := context.WithTimeout(ctx, promptTimeout)
	defer cancel()
	// Ctrl+C dismisses the question instead of killing a command that
	// succeeded.
	actx, stop := signal.NotifyContext(actx, os.Interrupt)
	defer stop()
	ans, err := cmdio.AskCancellable(actx, "Install Databricks AI tools for "+strings.Join(names, ", ")+"? [Y/n]", "")
	if err != nil {
		log.Debugf(ctx, "auto-install: skipped: the prompt was not answered: %v", err)
		// End the question's line.
		cmdio.LogString(ctx, "")
		r.settle(ctx, protos.AitoolsAutoInstallOutcomeUnanswered)
		return false
	}
	switch strings.ToLower(strings.TrimSpace(ans)) {
	case "", "y", "yes":
		return true
	case "n", "no":
		r.decline(ctx)
		return false
	}
	log.Debugf(ctx, "auto-install: skipped: the answer was neither yes nor no")
	r.settle(ctx, protos.AitoolsAutoInstallOutcomeUnanswered)
	cmdio.LogString(ctx, "Skipped. To install Databricks AI tools later, run: databricks aitools install")
	return false
}

// decline stores a "no" at the prompt as the opt-out, so no later version
// asks again.
func (r autoInstallRun) decline(ctx context.Context) {
	r.settle(ctx, protos.AitoolsAutoInstallOutcomeDeclined)
	if err := autoinstall.Decline(ctx); err != nil {
		log.Warnf(ctx, "Could not save the answer, so a later CLI version asks again: %v", err)
		cmdio.LogString(ctx, "Skipped. To install Databricks AI tools later, run: databricks aitools install")
		return
	}
	cmdio.LogString(ctx, "Skipped. You won't be asked again; to install Databricks AI tools later, run: databricks aitools install")
}

// skipOutcomes are the reported outcomes of the skips that settle a version.
var skipOutcomes = map[autoinstall.Reason]protos.AitoolsAutoInstallOutcome{
	autoinstall.ReasonSkippedCI:      protos.AitoolsAutoInstallOutcomeSkippedCI,
	autoinstall.ReasonSkippedOptOut:  protos.AitoolsAutoInstallOutcomeSkippedOptOut,
	autoinstall.ReasonSkippedHoldout: protos.AitoolsAutoInstallOutcomeSkippedHoldout,
}

// autoInstallRun is what the outcome of one auto-install run is reported with.
type autoInstallRun struct {
	version     string
	id          string
	arm         protos.AitoolsHoldoutArm
	interactive bool
}

// settleFailed records the version as done and reports a failed attempt.
func (r autoInstallRun) settleFailed(ctx context.Context, err error) {
	if recErr := autoinstall.Record(ctx, r.version, 0, r.id); recErr != nil {
		log.Debugf(ctx, "auto-install: could not save the state: %v", recErr)
	}
	r.report(ctx, protos.AitoolsAutoInstallOutcomeFailed, classifyInstallError(err))
}

// settle records the version as done and reports outcome.
func (r autoInstallRun) settle(ctx context.Context, outcome protos.AitoolsAutoInstallOutcome) {
	log.Debugf(ctx, "auto-install: %s", strings.ToLower(string(outcome)))
	if err := autoinstall.Record(ctx, r.version, 0, r.id); err != nil {
		log.Debugf(ctx, "auto-install: could not save the state: %v", err)
	}
	r.report(ctx, outcome, protos.AitoolsErrorCategoryUnspecified)
}

// report buffers the auto-install event; cmd/root uploads it at exit when the
// command resolved auth.
func (r autoInstallRun) report(ctx context.Context, outcome protos.AitoolsAutoInstallOutcome, category protos.AitoolsErrorCategory) {
	if category == protos.AitoolsErrorCategoryUnspecified {
		category = ""
	}
	telemetry.Log(ctx, protos.DatabricksCliLog{
		AitoolsAutoInstallEvent: &protos.AitoolsAutoInstallEvent{
			Outcome:       outcome,
			HoldoutArm:    r.arm,
			Interactive:   r.interactive,
			ErrorCategory: category,
		},
	})
}

func holdoutArm(bucket int) protos.AitoolsHoldoutArm {
	if autoinstall.InHoldout(bucket) {
		return protos.AitoolsHoldoutArmHoldout
	}
	return protos.AitoolsHoldoutArmTreatment
}

// installResult is what one auto-install attempt did.
type installResult struct {
	// noAgents is set when no supported agent has a config directory.
	noAgents bool
	// updated is set when an existing installation was updated.
	updated bool
	// agents names the agents that got new or updated tools.
	agents []string
}

// toolsPlan is what an auto-install attempt does: update the existing global
// installation, or install for the agents in plan.
type toolsPlan struct {
	globalDir string
	existing  *installer.InstallState
	plan      []agentPlanItem
	// keep, when set, is why the installed tools are left as they are.
	keep string
}

// planTools plans what `databricks aitools install --scope=global` runs
// without prompts, for the agents whose config directory exists. An existing
// global installation is updated as `databricks aitools update --scope=global`
// does instead, which keeps its downgrade guard, and keeps the skills whose
// files the user modified at their version. Planning reads only local state.
func planTools(ctx context.Context) (toolsPlan, error) {
	globalDir, err := installer.GlobalSkillsDir(ctx)
	if err != nil {
		return toolsPlan{}, err
	}
	existing, err := installer.LoadState(globalDir)
	if err != nil {
		return toolsPlan{}, err
	}
	if existing != nil {
		if !agentCLIsStoppable {
			// The update runs agent CLIs to update recorded plugins and to
			// migrate raw skills to plugins.
			candidates, err := pluginMigrationCandidatesForScope(ctx, installer.ScopeGlobal, existing)
			if err != nil {
				return toolsPlan{}, err
			}
			if len(existing.Plugins) > 0 || len(candidates) > 0 {
				return toolsPlan{keep: "the update would run agent CLIs"}, nil
			}
		}
		return toolsPlan{globalDir: globalDir, existing: existing}, nil
	}
	// Skills without install state have no checksums to tell the user's edits
	// from ours, and an install would replace them.
	if installer.HasUntrackedSkills(ctx, installer.ScopeGlobal) {
		return toolsPlan{keep: "skills without install state are on disk"}, nil
	}
	plan := buildPlan(ctx, agents.DetectInstalled(ctx), installer.ScopeGlobal, false, false)
	if !agentCLIsStoppable {
		plan = slices.DeleteFunc(plan, func(it agentPlanItem) bool {
			return it.delivery == deliveryPlugin && !it.cliMissing
		})
	}
	return toolsPlan{globalDir: globalDir, plan: plan}, nil
}

// newAgents names the agents a fresh install gives tools to: skills, or a
// plugin whose CLI is on PATH. An agent that keeps the plugin it has gets
// nothing. It is empty when an existing installation is updated.
func (t toolsPlan) newAgents() []string {
	if t.existing != nil {
		return nil
	}
	var names []string
	for _, it := range t.plan {
		if it.delivery == deliverySkills || (it.delivery == deliveryPlugin && !it.cliMissing) {
			names = append(names, it.agent.DisplayName)
		}
	}
	return names
}

// agentCLIsStoppable reports whether a timeout stops an agent CLI together
// with its descendants. On Windows it stops only the CLI, and a descendant
// left running could still be installing when the next run takes the lock,
// so auto-install there runs no agent CLI.
var agentCLIsStoppable = runtime.GOOS != "windows"

// install carries out the plan.
func (t toolsPlan) install(ctx context.Context) (installResult, error) {
	if t.keep != "" {
		log.Debugf(ctx, "auto-install: kept the installed tools: %s", t.keep)
		return installResult{}, nil
	}
	if t.existing != nil {
		return updateTools(ctx, t.globalDir, t.existing)
	}
	if len(t.plan) == 0 {
		return installResult{noAgents: true}, nil
	}
	opts := installer.InstallOptions{Scope: installer.ScopeGlobal}
	outcomes, err := executePlan(ctx, &installer.GitHubManifestSource{}, t.plan, opts, installInvocation{scope: installer.ScopeGlobal}, true)
	topErr := topLevelFailure(err)
	logInstallEvent(ctx, t.plan, installOpts{Scope: opts.Scope, Trigger: protos.AitoolsInstallTriggerAutoFirstRun}, classifyInstallError(topErr), outcomes)
	if err != nil && topErr == nil {
		// Reported per agent: take the category from the outcomes, as install does.
		err = withCategory(err, firstFailureCategory(outcomes))
	}
	var names []string
	for _, o := range outcomes {
		if o.status == outcomeInstalled {
			names = append(names, o.agent.DisplayName)
		}
	}
	return installResult{agents: names}, err
}

// updateTools runs the update command's body for the global scope. Its funnel
// events and its wait for telemetry auth belong to the command, so they are
// detached: the auto-install event reports the run.
func updateTools(ctx context.Context, globalDir string, before *installer.InstallState) (installResult, error) {
	uctx := telemetry.WithNewLogger(ctx)
	uctx = context.WithValue(uctx, funnelKey{}, &commandFunnel{command: protos.AitoolsCommandUpdate})
	uctx = context.WithValue(uctx, telemetryAuthKey{}, &telemetryAuth{})
	if err := runUpdate(uctx, updateFlags{scope: installer.ScopeGlobal, keepModified: true}); err != nil {
		return installResult{}, err
	}
	after, err := installer.LoadState(globalDir)
	if err != nil {
		return installResult{}, err
	}
	result := installResult{updated: true}
	if after != nil && toolsChanged(before, after) {
		for _, e := range buildAgentEntries(ctx, map[string]*installer.InstallState{installer.ScopeGlobal: after}) {
			if _, ok := e.Installed[installer.ScopeGlobal]; ok {
				result.agents = append(result.agents, e.DisplayName)
			}
		}
	}
	return result, nil
}

// toolsChanged reports whether an update changed the installed skills or
// plugin versions.
func toolsChanged(before, after *installer.InstallState) bool {
	return before.Release != after.Release ||
		!maps.Equal(before.Skills, after.Skills) ||
		!maps.EqualFunc(before.Plugins, after.Plugins, func(a, b installer.PluginRecord) bool { return a.Version == b.Version })
}

// quietContext discards what the installer prints and logs its warnings at
// debug level, so auto-install shows only its notice and its failures appear
// only in the debug log.
func quietContext(ctx context.Context) context.Context {
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText, nil, io.Discard, io.Discard, "", ""))
	return log.NewContext(ctx, slog.New(debugOnlyHandler{log.GetLogger(ctx).Handler()}))
}
