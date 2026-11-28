package aitools

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/databricks/cli/libs/aitools/autoinstall"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

// promptTimeout bounds the wait for an answer at the question. Tests lower it.
var promptTimeout = time.Minute

// newID is autoinstall.NewID; tests replace it.
var newID = autoinstall.NewID

// enterAgrees makes Enter, an empty reply, a "yes" at the question. The hint
// of the question shows it (see consentHint).
const enterAgrees = true

// skippedNotice is printed when the question was answered with neither yes nor
// no, or the answer could not be saved.
const skippedNotice = "Skipped. To install Databricks AI tools later, run: databricks aitools install"

// loginAnswerKey is the context key of the loginAnswer that AskAtLogin puts
// in the context of its command for AutoInstall.
type loginAnswerKey struct{}

// loginAnswerOf returns the answer to the question that the command of ctx
// asked, if it asked.
func loginAnswerOf(ctx context.Context) loginAnswer {
	a, _ := ctx.Value(loginAnswerKey{}).(loginAnswer)
	return a
}

// AskAtLogin asks the user of `databricks auth login`, after the login saved
// the profile, whether to install Databricks AI tools for the coding agents
// on this machine and keep them up to date (see autoinstall.AskAtLogin).
// "Yes" stores the consent, so AutoInstall, which the root command runs after
// the login, installs them, and says so when that fails. "No" stores the
// opt-out. Any other end leaves the consent unknown; the question is never
// asked again either way. It never fails the login and never writes to
// stdout.
//
// Ctrl+C at the question is recorded as an unanswered question, and
// AutoInstall then returns the signal without installing, so the root command
// ends the CLI by it.
func AskAtLogin(cmd *cobra.Command) {
	ctx := cmd.Context()
	defer func() {
		if r := recover(); r != nil {
			log.Debugf(ctx, "auto-install: panic at the question: %v", r)
		}
	}()
	if a := askAtLogin(ctx, skipsAutoInstall(cmd)); a.asked {
		cmd.SetContext(context.WithValue(ctx, loginAnswerKey{}, a))
	}
}

// loginAnswer is what the question at login got.
type loginAnswer struct {
	// asked is set when the question was shown.
	asked bool
	// answer is the user's answer: ConsentUnknown when the question was not
	// answered (it ran out of time, stdin was closed, Ctrl+C was pressed, or
	// the reply was neither yes nor no). It is stored unless saving it failed
	// (see stored), and a "yes" does not replace an opt-out that was stored
	// while the question waited.
	answer autoinstall.Consent
	// previous is the answer that was stored before.
	previous autoinstall.Consent
	// stored is set when the answer was stored as the consent or the opt-out.
	stored bool
	// sig is the signal that ended the question, if one did.
	sig os.Signal
}

// agreed reports whether the user agreed at the question and the consent was
// stored.
func (a loginAnswer) agreed() bool {
	return a.stored && a.answer == autoinstall.ConsentGiven
}

// askAtLogin asks the question of AskAtLogin when all of these hold: the user
// can see and answer it, no coding agent runs the CLI, the run is eligible
// (see autoinstall.Eligible), the user never answered and was never asked, a
// first install would give tools to at least one coding agent, and both
// gateOn and asksMachine allow asking for this version and machine. If gateOn
// fails, it asks nothing and records nothing.
func askAtLogin(ctx context.Context, skippedCommand bool) loginAnswer {
	// Most logins stop here, before anything is read.
	if !cmdio.IsPromptSupported(ctx) || useragent.AgentProvider() != "" {
		return loginAnswer{}
	}
	in := autoInstallInputs(ctx, skippedCommand)
	if !autoinstall.Eligible(in) {
		return loginAnswer{}
	}
	state, err := autoinstall.Load(ctx)
	if err != nil {
		log.Debugf(ctx, "auto-install: not asking: %v", err)
		return loginAnswer{}
	}
	if state.Consent() != autoinstall.ConsentUnknown || state.ConsentAsked {
		return loginAnswer{}
	}
	tools, err := planTools(quietContext(ctx), state)
	if err != nil {
		log.Debugf(ctx, "auto-install: not asking: %v", err)
		return loginAnswer{}
	}
	names := tools.newAgents()
	if len(names) == 0 {
		log.Debugf(ctx, "auto-install: not asking: a first install would give no coding agent the tools")
		return loginAnswer{}
	}
	// A stored ID decides the holdout without the gate lookup.
	if state.ID != "" && !asksMachine(state.ID) {
		log.Debugf(ctx, "auto-install: not asking: the machine is in the holdout")
		return loginAnswer{}
	}
	gateCtx, cancel := context.WithTimeout(ctx, gateTimeout)
	on, err := gateOn(gateCtx, in.Version)
	cancel()
	if err != nil {
		log.Debugf(ctx, "auto-install: not asking: could not resolve the gate: %v", err)
		return loginAnswer{}
	}
	if !on {
		log.Debugf(ctx, "auto-install: not asking: %s", autoinstall.ReasonGateOff)
		return loginAnswer{}
	}
	if state.ID == "" {
		// The holdout is the bucket of the machine's ID. It is stored, so that
		// the run after a "yes" and every later login use the same bucket.
		id, err := autoinstall.SaveID(ctx, newID())
		if err != nil {
			log.Debugf(ctx, "auto-install: not asking: could not save the state: %v", err)
			return loginAnswer{}
		}
		if !asksMachine(id) {
			log.Debugf(ctx, "auto-install: not asking: the machine is in the holdout")
			return loginAnswer{}
		}
	}
	return askConsent(ctx, names)
}

// asksMachine reports whether the question is asked on the machine with the
// ID id. A machine in the holdout never auto-installs, so it is not asked.
func asksMachine(id string) bool {
	return !autoinstall.InHoldout(autoinstall.Bucket(id))
}

// askConsent asks whether to install for the agents names and stores the
// answer. Only what is typed after the question is shown answers it: what
// was typed before, such as the next command typed while the login ran or an
// Enter pressed while recording the question waited for the state lock, is
// discarded unread once the question is shown, right before the answer is
// read. Where pending input cannot be discarded, the question is not asked;
// that is checked first, so such a machine records nothing. The question is
// recorded as asked before it is shown, so it is asked once even when it is
// never answered: the process is killed, stdin is closed, Ctrl+C is pressed,
// nobody answers within promptTimeout, or the answer is neither yes nor no.
func askConsent(ctx context.Context, names []string) loginAnswer {
	if err := cmdio.DiscardInput(ctx); err != nil {
		log.Debugf(ctx, "auto-install: not asking: could not discard the pending input: %v", err)
		return loginAnswer{}
	}
	marked, err := autoinstall.MarkConsentAsked(ctx)
	if err != nil {
		log.Debugf(ctx, "auto-install: not asking: could not save the state: %v", err)
		return loginAnswer{}
	}
	if !marked {
		// Another CLI process asked, or the user answered, since the state
		// was read.
		return loginAnswer{}
	}
	a := loginAnswer{asked: true}
	actx, cancel := context.WithTimeout(ctx, promptTimeout)
	defer cancel()
	// Ctrl+C ends the question. It stays unanswered, and the signal is
	// returned for the CLI to end by.
	actx, stop := notifyInterrupt(actx, os.Interrupt)
	// Stops catching Ctrl+C even after a panic; stopping again does nothing.
	defer stop()
	reply, err := cmdio.AskCancellableAfterQuestion(actx, "Install Databricks AI tools for "+strings.Join(names, ", ")+" and keep them up to date? "+consentHint(), "")
	a.sig = stop()
	if err == nil && a.sig != nil {
		// Ctrl+C right after an answer: the user asked the CLI to stop.
		err = errInterrupted
	}
	if err != nil {
		log.Debugf(ctx, "auto-install: the question was not answered: %v", err)
		// End the question's line.
		cmdio.LogString(ctx, "")
		return a
	}
	a.answer = parseConsentAnswer(reply)
	switch a.answer {
	case autoinstall.ConsentGiven:
		a.previous, err = autoinstall.GiveConsentIfUnknown(ctx)
		switch {
		case err != nil:
			log.Warnf(ctx, "Could not save the answer: %v", err)
			cmdio.LogString(ctx, skippedNotice)
		case a.previous == autoinstall.ConsentOptedOut:
			log.Debugf(ctx, "auto-install: kept the opt-out stored while the question waited")
		default:
			a.stored = true
		}
	case autoinstall.ConsentOptedOut:
		a.previous, a.stored = decline(ctx)
	default:
		log.Debugf(ctx, "auto-install: the answer was neither yes nor no")
		cmdio.LogString(ctx, skippedNotice)
	}
	return a
}

// decline stores a "no" to the question as the opt-out. It returns the answer
// stored before, and whether it stored the opt-out.
func decline(ctx context.Context) (previous autoinstall.Consent, stored bool) {
	previous, err := autoinstall.Decline(ctx)
	if err != nil {
		// This process honors the answer anyway (see autoinstall.Declined),
		// and the question is not asked again.
		log.Warnf(ctx, "Could not save the answer: %v", err)
		cmdio.LogString(ctx, skippedNotice)
		return previous, false
	}
	cmdio.LogString(ctx, "Skipped. You won't be asked again; to install Databricks AI tools later, run: databricks aitools install")
	return previous, true
}

// consentHint shows the answers to the question, with the one Enter gives in
// capitals.
func consentHint() string {
	if enterAgrees {
		return "[Y/n]"
	}
	return "[y/n]"
}

// parseConsentAnswer returns the answer that reply gives: ConsentUnknown for
// one that is neither yes nor no.
func parseConsentAnswer(reply string) autoinstall.Consent {
	switch strings.ToLower(strings.TrimSpace(reply)) {
	case "y", "yes":
		return autoinstall.ConsentGiven
	case "n", "no":
		return autoinstall.ConsentOptedOut
	case "":
		if enterAgrees {
			return autoinstall.ConsentGiven
		}
	}
	return autoinstall.ConsentUnknown
}
