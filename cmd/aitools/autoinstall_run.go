package aitools

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/log/handler"
	"github.com/databricks/cli/libs/process"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/versioncheck"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

const (
	// installTimeout bounds the install or update (its downloads and the
	// agent CLIs it runs). One that runs out of time counts as a failed
	// attempt and is retried on a later run.
	installTimeout = 10 * time.Second

	// promptTimeout bounds the wait for an answer at the install prompt.
	promptTimeout = time.Minute
)

// gateTimeout bounds the gate lookup, one compatibility manifest fetch. Every
// machine pays it once per version while the gate is off, and a machine that
// cannot reach the manifest waits this long after its command finished, so it
// is kept short. A version gets one lookup: one that fails is not repeated
// (see autoInstall). Tests raise it.
var gateTimeout = time.Second

// gateOn is autoinstall.GateOn; tests replace it.
var gateOn = autoinstall.GateOn

// now is time.Now; tests replace it.
var now = time.Now

// rootInUserHome is autoinstall.RootInUserHome; tests replace it.
var rootInUserHome = autoinstall.RootInUserHome

// inContainer is autoinstall.InContainer; tests replace it.
var inContainer = autoinstall.InContainer

// notifyInterrupt catches sigs: it returns a context that is canceled when
// the CLI receives one of them, and a function that stops catching them and
// returns the signal received, or nil. Tests replace it.
var notifyInterrupt = func(ctx context.Context, sigs ...os.Signal) (context.Context, func() os.Signal) {
	ctx, cancel := context.WithCancel(ctx)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, sigs...)
	var got os.Signal
	done := make(chan struct{})
	go func() {
		defer close(done)
		select {
		case got = <-ch:
			cancel()
		case <-ctx.Done():
		}
	}()
	return ctx, func() os.Signal {
		signal.Stop(ch)
		cancel()
		<-done
		if got != nil {
			return got
		}
		// One that arrived after ctx ended.
		select {
		case sig := <-ch:
			return sig
		default:
			return nil
		}
	}
}

// errInterrupted is why a run stopped when it caught a signal.
var errInterrupted = errors.New("interrupted")

// autoInstallDisable is the opt-out command named in the notice.
const autoInstallDisable = "databricks aitools auto-install disable"

// AutoInstall installs Databricks AI tools for the coding agents on this
// machine, or updates an existing installation, on the first run of a new CLI
// version. The root command runs it after a command succeeded. It never fails
// the command and never writes to stdout. On stderr it prints the install
// prompt (in a terminal) and one line on what it did; why it skipped or failed
// goes only to the debug log.
//
// It returns the signal that stopped it, if one did: Ctrl+C at the question,
// or Ctrl+C, a request to terminate or the terminal closing during the gate
// lookup. It records what it must first. The root command then ends the CLI
// by the signal, so that the shell stops the script, loop or command list
// that ran the command, as it would have without auto-install.
func AutoInstall(cmd *cobra.Command) os.Signal {
	ctx := cmd.Context()
	defer func() {
		if r := recover(); r != nil {
			log.Debugf(ctx, "auto-install: panic: %v", r)
		}
	}()
	return autoInstall(ctx, skipsAutoInstall(cmd))
}

// skipsAutoInstall reports whether cmd never triggers auto-install: the
// commands the new-release notice skips (`version` is how a user checks what
// was just installed; help and shell completion output is read by machines or
// shown on every keystroke), the credential helpers that other tools call for
// a token, and the aitools commands, which manage the tools explicitly.
//
// Runs without a terminal and runs with JSON output are not skipped, unlike
// for the notice: coding agents run the CLI that way, and they are what the
// tools are for. Auto-install writes nothing to stdout.
func skipsAutoInstall(cmd *cobra.Command) bool {
	if versioncheck.IsExemptCommand(cmd) || agents.IsCredentialHelperCommand(cmd) {
		return true
	}
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "aitools" {
			return true
		}
	}
	return false
}

// autoInstallInputs returns the decision inputs that need neither the state
// nor the gate.
func autoInstallInputs(ctx context.Context, skippedCommand bool) autoinstall.Inputs {
	return autoinstall.Inputs{
		SkippedCommand: skippedCommand,
		OnRuntime:      dbr.RunsOnRuntime(ctx),
		CacheDisabled:  autoinstall.CacheDisabled(ctx),
		Version:        autoinstall.Version(ctx),
		RootInUserHome: rootInUserHome(ctx),
		CI:             agents.InCI(ctx),
		// A "no" earlier in this process counts even if storing it failed.
		EnvOptOut: autoinstall.EnvOptedOut(ctx) || autoinstall.Declined(),
		Now:       now(),
	}
}

func autoInstall(ctx context.Context, skippedCommand bool) os.Signal {
	in := autoInstallInputs(ctx, skippedCommand)
	version := in.Version
	// What never auto-installs skips before the state is read, and without a
	// log line: these are most runs. Root in a user's home is rare, and a user
	// who always runs the CLI under sudo needs a way to see why nothing
	// installs, so it logs.
	if reason := autoinstall.Decide(in); slices.Contains(stateFreeSkips, reason) {
		if reason == autoinstall.ReasonRootInUserHome {
			log.Debugf(ctx, "auto-install: skipped: %s", reason)
		}
		return nil
	}
	var err error
	in.State, err = autoinstall.Load(ctx)
	if err != nil {
		log.Debugf(ctx, "auto-install: skipped: %v", err)
		return nil
	}
	reason := autoinstall.Decide(in)
	if !reason.Due() {
		log.Debugf(ctx, "auto-install: skipped: %s", reason)
		return nil
	}
	// A run that would ask before installing but cannot skips before the lock
	// and the gate lookup, since it repeats on every such run until a run that
	// can ask. A holdout machine or a gate-off version would otherwise settle
	// after the lookup; they wait for that run too, so holdout and treatment
	// machines are counted alike.
	//
	// A machine with no coding agent skips there as well: the gate changes
	// nothing for it, and such machines (containers, job pods) often have no
	// route to the manifest and a home directory that keeps no state, so they
	// would wait for the lookup on every run. The version is not settled, so
	// an agent set up later is still found.
	if reason == autoinstall.ReasonGateOff {
		tools, err := planTools(quietContext(ctx), in.State)
		if err == nil && tools.skip == protos.AitoolsAutoInstallOutcomeSkippedNoAgents {
			log.Debugf(ctx, "auto-install: skipped: %s", tools.skipWhy)
			return nil
		}
		if err == nil && autoInstallAsks() && in.State.AcceptedVersion != version && !canAsk(ctx) && len(tools.newAgents()) > 0 {
			skipUnaskable(ctx, in.State.ID)
			return nil
		}
	}

	unlock, ok := autoinstall.TryLock(ctx)
	if !ok {
		log.Debugf(ctx, "auto-install: skipped: another CLI process is running it")
		return nil
	}
	defer unlock()
	// Another process may have finished a run since the state was read.
	in.State, err = autoinstall.Load(ctx)
	if err != nil {
		log.Debugf(ctx, "auto-install: skipped: %v", err)
		return nil
	}
	// Planning before the lock took time; the run is decided from here.
	in.Now = now()
	id := in.State.ID
	if id == "" {
		// Saved before it decides anything, so this run and one that cannot
		// ask (skipUnaskable, which does not hold this lock) use the same ID.
		id, err = autoinstall.SaveID(ctx, autoinstall.NewID())
		if err != nil {
			log.Debugf(ctx, "auto-install: skipped: could not save the state: %v", err)
			return nil
		}
	}
	in.Bucket = autoinstall.Bucket(id)

	// The gate costs a manifest lookup, so it is resolved only when it is
	// what decides the run.
	reason = autoinstall.Decide(in)
	if reason == autoinstall.ReasonGateOff {
		// Ctrl+C, a request to terminate or the terminal closing ends the
		// lookup. The version is recorded as for a failed lookup, and the
		// signal is returned for the CLI to end by.
		sigCtx, stop := notifyInterrupt(ctx, process.InterruptSignals()...)
		gateCtx, cancel := context.WithTimeout(sigCtx, gateTimeout)
		in.GateOn, err = gateOn(gateCtx, version)
		cancel()
		sig := stop()
		if err == nil && sig != nil {
			// An interrupted lookup can still answer, from a manifest cached
			// earlier. The user asked the CLI to stop, so the run ends here
			// like one whose lookup failed.
			err = errInterrupted
		}
		if err != nil {
			// The gate is unknown. A machine that cannot reach the manifest
			// (no route to GitHub) would wait for the lookup after every
			// command, so the version gets no other attempt; the next version
			// looks the gate up again. The feature may be off, so nothing is
			// reported.
			log.Debugf(ctx, "auto-install: could not resolve the gate, not trying again for this version: %v", err)
			// No install started, so there is no last attempt.
			if err := autoinstall.Record(ctx, version, autoinstall.MaxAttempts, id, time.Time{}); err != nil {
				log.Debugf(ctx, "auto-install: could not save the state: %v", err)
			}
			return sig
		}
		reason = autoinstall.Decide(in)
	}
	run := autoInstallRun{
		version:     version,
		id:          id,
		arm:         holdoutArm(in.Bucket),
		interactive: cmdio.HasTerminal(ctx) && autoInstallAsks(),
		now:         in.Now,
	}
	if !reason.Due() {
		// Another process settled the version or recorded a failed attempt
		// while this one waited.
		log.Debugf(ctx, "auto-install: skipped: %s", reason)
		return nil
	}
	if reason == autoinstall.ReasonGateOff {
		// The feature is not launched, so the skip is not reported.
		log.Debugf(ctx, "auto-install: skipped: %s", reason)
		if err := autoinstall.Record(ctx, version, 0, id, in.Now); err != nil {
			log.Debugf(ctx, "auto-install: could not save the state: %v", err)
		}
		return nil
	}
	if outcome, ok := skipOutcomes[reason]; ok {
		run.settle(ctx, outcome)
		return nil
	}
	return run.attempt(ctx, in.State)
}

// attempt installs or updates the tools and reports the outcome. prior is the
// state the run was decided on.
func (r autoInstallRun) attempt(ctx context.Context, prior autoinstall.State) os.Signal {
	attempt := autoinstall.NextAttempt(prior, r.version)
	qctx := quietContext(ctx)
	tools, err := planTools(qctx, prior)
	// A caller that is not a coding agent is asked before a first install,
	// once per version: a "yes" covers that version's retries. A planning
	// failure is a failed attempt, and its retry asks.
	asked := false
	if err == nil && autoInstallAsks() && prior.AcceptedVersion != r.version && len(tools.newAgents()) > 0 {
		// autoInstall skips this case before the lock; the state it decided
		// on may have changed since.
		if !canAsk(ctx) {
			skipUnaskable(ctx, r.id)
			return nil
		}
		if ok, sig := r.ask(ctx, tools.newAgents()); !ok {
			return sig
		}
		asked = true
	}
	// Count the attempt before making it: an attempt that never finishes (the
	// process is killed) still counts, and a state that cannot be saved stops
	// auto-install instead of retrying it on every run.
	started, recErr := autoinstall.StartAttempt(ctx, r.version, attempt, r.id, r.now)
	if recErr != nil {
		log.Debugf(ctx, "auto-install: skipped: could not save the state: %v", recErr)
		return nil
	}
	if !started {
		// The user opted out after this run read the state.
		r.settle(ctx, protos.AitoolsAutoInstallOutcomeSkippedOptOut)
		return nil
	}
	if err == nil && tools.fresh() {
		// Skills an unfinished install leaves have no install state; the
		// marker tells a later run that they are its own.
		err = autoinstall.MarkInstalling(ctx)
	}
	var result installResult
	if err == nil {
		runCtx, cancel := context.WithTimeout(qctx, installTimeout)
		result, err = tools.install(runCtx)
		cancel()
		if tools.fresh() && (err != nil || result.unfinished) {
			saveInstallStateDigest(ctx, tools.globalDir)
		}
	}
	if _, refused := errors.AsType[*installer.RefusedError](err); refused {
		// The installed skills are newer than the release this CLI would
		// install (the downgrade guard), or a skills agent's directory has an
		// entry named after a skill to install that the user or another tool
		// put there. Retrying this version changes nothing: a later version
		// tries again, and installs once a newer release is out or the user
		// has moved that entry out of the agent's skills directory.
		//
		// The first install stays unfinished only when this attempt or an
		// earlier one left something behind: a refusal before anything was
		// written leaves nothing for a later run to finish, and the marker
		// would make it take skills without install state in the global
		// directory for its own.
		log.Debugf(ctx, "auto-install: kept the installed tools: %v", err)
		unfinished := tools.fresh() && (prior.Installing || installStateSaved(tools.globalDir))
		result, err = installResult{outcome: protos.AitoolsAutoInstallOutcomeKept, unfinished: unfinished, refusal: err}, nil
	}
	switch {
	case err != nil:
		outcome := protos.AitoolsAutoInstallOutcomeFailed
		if attempt >= autoinstall.MaxAttempts {
			outcome = protos.AitoolsAutoInstallOutcomeRetriesExhausted
		}
		log.Debugf(ctx, "auto-install: attempt %d of %d failed: %v", attempt, autoinstall.MaxAttempts, err)
		r.report(ctx, outcome, classifyInstallError(err))
		if asked {
			// The user is waiting for the answer to take effect.
			cmdio.LogString(ctx, "Could not install Databricks AI tools. To install them, run: databricks aitools install")
		}
	default:
		if result.unfinished {
			r.settleUnfinished(ctx, result.outcome)
		} else {
			r.settle(ctx, result.outcome)
		}
		if result.outcome == protos.AitoolsAutoInstallOutcomeInstalled && len(result.agents) > 0 {
			verb := "Installed"
			if result.updated {
				verb = "Updated"
			}
			cmdio.LogString(ctx, fmt.Sprintf("%s Databricks AI tools for %s. Disable: %s", verb, strings.Join(result.agents, ", "), autoInstallDisable))
		}
		if asked && result.refusal != nil {
			// The user said yes and waits for the skills; the refusal names
			// what was kept and the command that replaces it.
			cmdio.LogString(ctx, "Did not install the Databricks AI tools skills: "+result.refusal.Error())
		}
	}
	return nil
}

// installStateSaved reports whether the global directory has install state.
// State that cannot be read counts as saved: something wrote it.
func installStateSaved(globalDir string) bool {
	existing, err := installer.LoadState(globalDir)
	return err != nil || existing != nil
}

// saveInstallStateDigest records the digest of the install state that a first
// install that did not finish saved, so that a later run can tell it from
// install state that anything else wrote. Without the digest, a later run
// updates that state instead of finishing the install.
func saveInstallStateDigest(ctx context.Context, globalDir string) {
	existing, err := installer.LoadState(globalDir)
	if err == nil && existing == nil {
		return
	}
	var digest string
	if err == nil {
		digest, err = installStateDigest(existing)
	}
	if err == nil {
		err = autoinstall.SaveInstallStateDigest(ctx, digest)
	}
	if err != nil {
		log.Debugf(ctx, "auto-install: could not save the state: %v", err)
	}
}

// autoInstallAsks reports whether auto-install asks before a first install:
// no coding agent is detected. A run that cannot ask (see canAsk) skips;
// coding agents install without a question.
func autoInstallAsks() bool {
	return useragent.AgentProvider() == ""
}

// canAsk reports whether the question can be seen and answered: see
// cmdio.IsLinePromptSupported. A CLI process that the CLI started itself does
// not ask either; the parent asks after its own command. Nor does a run in a
// container: `docker run -t` without `-i` gives it a terminal that never
// delivers an answer, and a container's home often keeps no state, so the
// unanswered question would wait promptTimeout on every run.
func canAsk(ctx context.Context) bool {
	return cmdio.IsLinePromptSupported(ctx) && !agents.IsSelfExec(ctx) && !inContainer()
}

// skipUnaskable skips a run that would ask before installing but cannot: the
// question could not be seen or answered safely, and installing without it
// would skip the consent. The version is not settled, so a later run that can
// ask does. A machine with no stored ID gets one, so that such runs do not
// draw a new holdout bucket each time.
func skipUnaskable(ctx context.Context, storedID string) {
	log.Debugf(ctx, "auto-install: skipped: cannot ask (a stream is not a terminal, the CLI runs in the background, in a container or for another CLI process, or stdin is in use)")
	if storedID != "" {
		return
	}
	if _, err := autoinstall.SaveID(ctx, autoinstall.NewID()); err != nil {
		log.Debugf(ctx, "auto-install: could not save the state: %v", err)
	}
}

// skippedNotice is printed when the question was answered with neither yes nor
// no, or the "no" could not be saved.
const skippedNotice = "Skipped. To install Databricks AI tools later, run: databricks aitools install"

// ask asks whether to install for the agents names and reports whether to go
// ahead, and the signal that ended the question, if one did (see
// AutoInstall). What was typed before the question, such as the next command
// typed while this one ran, is discarded unread; where it cannot be (a pipe
// that Git Bash's mintty counts as a terminal), the question is not asked.
// The version is settled before asking, so the question is asked once per
// version even when it is never answered: the process is killed, stdin is
// closed, Ctrl+C is pressed, nobody answers within promptTimeout (an
// unrecognized agent in a pseudo-terminal), or the answer is neither yes nor
// no. "Yes" is recorded so that the version's retries do not ask again; only
// "no" stores the opt-out.
func (r autoInstallRun) ask(ctx context.Context, names []string) (bool, os.Signal) {
	if err := cmdio.DiscardInput(ctx); err != nil {
		log.Debugf(ctx, "auto-install: could not discard the pending input: %v", err)
		skipUnaskable(ctx, r.id)
		return false, nil
	}
	if err := autoinstall.Record(ctx, r.version, 0, r.id, r.now); err != nil {
		log.Debugf(ctx, "auto-install: skipped: could not save the state: %v", err)
		return false, nil
	}
	actx, cancel := context.WithTimeout(ctx, promptTimeout)
	defer cancel()
	// Ctrl+C ends the question. The version is settled as unanswered, and
	// the signal is returned for the CLI to end by.
	actx, stop := notifyInterrupt(actx, os.Interrupt)
	ans, err := cmdio.AskCancellable(actx, "Install Databricks AI tools for "+strings.Join(names, ", ")+"? [Y/n]", "")
	sig := stop()
	if err == nil && sig != nil {
		// Ctrl+C right after an answer: the user asked the CLI to stop.
		err = errInterrupted
	}
	if err != nil {
		log.Debugf(ctx, "auto-install: skipped: the prompt was not answered: %v", err)
		// End the question's line.
		cmdio.LogString(ctx, "")
		r.settle(ctx, protos.AitoolsAutoInstallOutcomeUnanswered)
		return false, sig
	}
	switch strings.ToLower(strings.TrimSpace(ans)) {
	case "", "y", "yes":
		if err := autoinstall.Accept(ctx, r.version); err != nil {
			// The install goes ahead; a retry of it asks again.
			log.Debugf(ctx, "auto-install: could not save the answer: %v", err)
		}
		return true, nil
	case "n", "no":
		r.decline(ctx)
		return false, nil
	}
	log.Debugf(ctx, "auto-install: skipped: the answer was neither yes nor no")
	r.settle(ctx, protos.AitoolsAutoInstallOutcomeUnanswered)
	cmdio.LogString(ctx, skippedNotice)
	return false, nil
}

// decline stores a "no" at the prompt as the opt-out, so no later version
// asks again.
func (r autoInstallRun) decline(ctx context.Context) {
	r.settle(ctx, protos.AitoolsAutoInstallOutcomeDeclined)
	if err := autoinstall.Decline(ctx); err != nil {
		log.Warnf(ctx, "Could not save the answer, so a later CLI version asks again: %v", err)
		cmdio.LogString(ctx, skippedNotice)
		return
	}
	cmdio.LogString(ctx, "Skipped. You won't be asked again; to install Databricks AI tools later, run: databricks aitools install")
}

// stateFreeSkips are the reasons Decide finds without the state.
var stateFreeSkips = []autoinstall.Reason{
	autoinstall.ReasonSkippedCommand,
	autoinstall.ReasonOnRuntime,
	autoinstall.ReasonCacheDisabled,
	autoinstall.ReasonDevelopmentBuild,
	autoinstall.ReasonRootInUserHome,
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
	// now is the time the run was decided at.
	now time.Time
}

// settle records the version as done and reports outcome.
func (r autoInstallRun) settle(ctx context.Context, outcome protos.AitoolsAutoInstallOutcome) {
	log.Debugf(ctx, "auto-install: %s", strings.ToLower(string(outcome)))
	if err := autoinstall.Record(ctx, r.version, 0, r.id, r.now); err != nil {
		log.Debugf(ctx, "auto-install: could not save the state: %v", err)
	}
	r.report(ctx, outcome, protos.AitoolsErrorCategoryUnspecified)
}

// settleUnfinished settles the version as settle does, but keeps the first
// install unfinished: it was refused the skills, so a later version finishes
// it as a first install instead of updating what it recorded.
func (r autoInstallRun) settleUnfinished(ctx context.Context, outcome protos.AitoolsAutoInstallOutcome) {
	log.Debugf(ctx, "auto-install: %s, the first install is unfinished", strings.ToLower(string(outcome)))
	if err := autoinstall.RecordUnfinished(ctx, r.version, r.id); err != nil {
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
	// outcome is the reported outcome of an attempt that did not fail.
	outcome protos.AitoolsAutoInstallOutcome
	// updated is set when an existing installation was updated.
	updated bool
	// agents names the agents that got new or updated tools.
	agents []string
	// unfinished is set when a first install was refused the skills after
	// it left something behind: an agent did not get its tools.
	unfinished bool
	// refusal is why the skills were refused, if they were.
	refusal error
}

// toolsPlan is what an auto-install attempt does: update the existing global
// installation, install for the agents in plan, or leave the tools alone.
type toolsPlan struct {
	globalDir string
	existing  *installer.InstallState
	plan      []agentPlanItem
	// skip, when set, is the outcome of a plan that changes nothing, and
	// skipWhy says why.
	skip    protos.AitoolsAutoInstallOutcome
	skipWhy string
}

// fresh reports whether the plan is a first install.
func (t toolsPlan) fresh() bool {
	return t.skip == "" && t.existing == nil && len(t.plan) > 0
}

// planTools plans what `databricks aitools install --scope=global` runs
// without prompts, for the agents whose config directory exists. An existing
// global installation is updated as `databricks aitools update --scope=global`
// does instead, which keeps its downgrade guard, also keeps an installed
// release newer than a pinned one, keeps the skills whose files the user
// modified at their version, and does not migrate skills to plugins.
// A first install of auto-install that did not finish is planned again as a
// first install, even when it saved install state, as long as nothing else
// rewrote that state since (see autoinstall.State.InstallStateDigest): update
// reinstalls only the skills whose version changed, so it would never give the
// skills to an agent that did not get them. Planning reads only local state;
// prior is the auto-install state.
func planTools(ctx context.Context, prior autoinstall.State) (toolsPlan, error) {
	globalDir, err := installer.GlobalSkillsDir(ctx)
	if err != nil {
		return toolsPlan{}, err
	}
	existing, err := installer.LoadState(globalDir)
	if err != nil {
		return toolsPlan{}, err
	}
	// What an unfinished first install of auto-install left is its own: skills
	// without install state, or install state nothing else rewrote since.
	unfinished := prior.Installing
	if unfinished && existing != nil {
		digest, err := installStateDigest(existing)
		if err != nil {
			return toolsPlan{}, err
		}
		unfinished = digest == prior.InstallStateDigest
	}
	if existing != nil && !unfinished {
		// The update runs agent CLIs to update the recorded plugins.
		if !agentCLIsStoppable && len(existing.Plugins) > 0 {
			return toolsPlan{skip: protos.AitoolsAutoInstallOutcomeSkippedPlatform, skipWhy: "the update would run agent CLIs"}, nil
		}
		return toolsPlan{globalDir: globalDir, existing: existing}, nil
	}
	// Skills without install state have no checksums to tell the user's edits
	// from ours, and an install would replace them.
	if !unfinished && installer.HasUntrackedSkills(ctx, installer.ScopeGlobal) {
		return toolsPlan{skip: protos.AitoolsAutoInstallOutcomeKept, skipWhy: "skills without install state are on disk"}, nil
	}
	plan := buildPlan(ctx, agents.DetectInstalled(ctx), installer.ScopeGlobal, false, false)
	if len(plan) == 0 {
		return toolsPlan{skip: protos.AitoolsAutoInstallOutcomeSkippedNoAgents, skipWhy: "no supported coding agent has a config directory"}, nil
	}
	if unfinished {
		// An agent that got its plugin before the install stopped has its
		// tools: installing the plugin again would rerun its CLI and announce
		// the agent on every version until the install finishes.
		plan = slices.DeleteFunc(plan, func(it agentPlanItem) bool {
			return it.delivery == deliveryPlugin && pluginAlreadyInstalledFn(ctx, it.agent, installer.ScopeGlobal)
		})
		if len(plan) == 0 {
			return toolsPlan{skip: protos.AitoolsAutoInstallOutcomeUpToDate, skipWhy: "every agent got its tools before the install stopped"}, nil
		}
	}
	if !agentCLIsStoppable {
		plan = slices.DeleteFunc(plan, func(it agentPlanItem) bool {
			return it.delivery == deliveryPlugin && !it.cliMissing
		})
		if len(plan) == 0 {
			return toolsPlan{skip: protos.AitoolsAutoInstallOutcomeSkippedPlatform, skipWhy: "every agent installs through its CLI"}, nil
		}
	}
	return toolsPlan{globalDir: globalDir, plan: plan}, nil
}

// installStateDigest returns a digest of the install state without its MCP
// section, which the mcp commands write.
func installStateDigest(s *installer.InstallState) (string, error) {
	c := *s
	c.MCP = nil
	data, err := json.Marshal(&c)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// newAgents names the agents a fresh install gives tools to: skills, or a
// plugin whose CLI is on PATH. An agent that keeps the plugin it has gets
// nothing. It is empty unless the plan is a first install (see fresh).
func (t toolsPlan) newAgents() []string {
	if !t.fresh() {
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
	if t.skip != "" {
		log.Debugf(ctx, "auto-install: nothing to do: %s", t.skipWhy)
		return installResult{outcome: t.skip}, nil
	}
	if t.existing != nil {
		return updateTools(ctx, t.globalDir, t.existing)
	}
	// A skill in an agent's own directory, named after a skill the install
	// gives that agent, that is neither a link to nor a copy of the skill in
	// the global directory was put there by the user or another tool, and the
	// install would move it to a temporary directory. The unfinished-install
	// marker does not make it ours: the user may have added it after the
	// install stopped.
	opts := installer.InstallOptions{Scope: installer.ScopeGlobal, KeepHandPlacedSkills: true}
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
	// executePlan installs the skills before the plugins, and the skills
	// refusal does not stop the plugin installs: it is returned after them.
	// So a refusal still reports the agents that got a plugin, and leaves the
	// install unfinished for the skills agents. Without any such agent the
	// caller reports it as kept.
	if _, refused := errors.AsType[*installer.RefusedError](err); refused && len(names) > 0 {
		log.Debugf(ctx, "auto-install: kept the skills: %v", err)
		return installResult{outcome: protos.AitoolsAutoInstallOutcomeInstalled, agents: names, unfinished: true, refusal: err}, nil
	}
	return installResult{outcome: protos.AitoolsAutoInstallOutcomeInstalled, agents: names}, err
}

// updateTools runs the update command's body for the global scope. Its funnel
// events and its wait for telemetry auth belong to the command, so they are
// detached: the auto-install event reports the run.
func updateTools(ctx context.Context, globalDir string, before *installer.InstallState) (installResult, error) {
	uctx := telemetry.WithNewLogger(ctx)
	uctx = context.WithValue(uctx, funnelKey{}, &commandFunnel{command: protos.AitoolsCommandUpdate})
	uctx = context.WithValue(uctx, telemetryAuthKey{}, &telemetryAuth{})
	updateErr := runUpdate(uctx, updateFlags{scope: installer.ScopeGlobal, keepModified: true})
	// The update refuses the raw skills after it updated the plugins, so a
	// refusal still reports the plugins it changed.
	_, refused := errors.AsType[*installer.RefusedError](updateErr)
	if updateErr != nil && !refused {
		return installResult{}, updateErr
	}
	after, err := installer.LoadState(globalDir)
	if err != nil {
		return installResult{}, err
	}
	result := installResult{outcome: protos.AitoolsAutoInstallOutcomeUpToDate, updated: true}
	if after != nil {
		result.agents = changedAgents(ctx, before, after)
	}
	if len(result.agents) == 0 {
		// Nothing changed: the caller reports a refusal as kept.
		return result, updateErr
	}
	if refused {
		log.Debugf(ctx, "auto-install: kept the installed skills: %v", updateErr)
	}
	result.outcome = protos.AitoolsAutoInstallOutcomeInstalled
	return result, nil
}

// changedAgents names the agents an update changed: those with skills when a
// skill version changed, and those whose plugin version changed. The release
// is not compared: it advances even when every outdated skill was kept.
func changedAgents(ctx context.Context, before, after *installer.InstallState) []string {
	skillsChanged := !maps.Equal(before.Skills, after.Skills)
	var names []string
	for _, e := range buildAgentEntries(ctx, map[string]*installer.InstallState{installer.ScopeGlobal: after}) {
		info, ok := e.Installed[installer.ScopeGlobal]
		if !ok {
			continue
		}
		changed := skillsChanged
		if info.Delivery == deliveryPlugin.String() {
			changed = before.Plugins[e.Name].Version != after.Plugins[e.Name].Version
		}
		if changed {
			names = append(names, e.DisplayName)
		}
	}
	return names
}

// quietContext discards what the installer prints and logs its warnings at
// debug level, so auto-install shows only its notice and its failures appear
// only in the debug log.
func quietContext(ctx context.Context) context.Context {
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText, nil, io.Discard, io.Discard, "", ""))
	return log.NewContext(ctx, slog.New(handler.DebugOnlyHandler{Handler: log.GetLogger(ctx).Handler()}))
}
