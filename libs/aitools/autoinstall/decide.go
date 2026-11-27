package autoinstall

import (
	"time"

	"github.com/databricks/cli/internal/build"
	"golang.org/x/mod/semver"
)

const (
	// MaxAttempts is how many failed attempts one CLI version gets. A new
	// version starts over.
	MaxAttempts = 10

	// firstRetryDelay is the wait after the first failed attempt of a
	// version before the next one. It doubles with each failed attempt, so
	// the last of MaxAttempts comes about 42 hours after the first.
	firstRetryDelay = 5 * time.Minute

	// HoldoutPercent is the share of machines (by Bucket) that never
	// auto-install, so the install rate can be compared with a control group.
	HoldoutPercent = 10
)

// Reason is what Decide concluded. The reasons that are reported are also the
// outcome values of the auto-install telemetry event.
type Reason string

const (
	// Not due: the run records and reports nothing.
	ReasonSkippedCommand   Reason = "skipped_command"
	ReasonOnRuntime        Reason = "on_runtime"
	ReasonCacheDisabled    Reason = "cache_disabled"
	ReasonDevelopmentBuild Reason = "development_build"
	ReasonRootInUserHome   Reason = "root_in_user_home"
	ReasonAlreadyRan       Reason = "already_ran"
	ReasonRetriesExhausted Reason = "retries_exhausted"
	ReasonRetryNotDue      Reason = "retry_not_due"
	// The user never agreed to installing the tools. The version is left
	// unsettled, so the first run after the user agrees installs.
	ReasonNoConsent Reason = "no_consent"

	// Settle the version without installing. All but ReasonGateOff are
	// reported: the feature is not launched while the gate is off.
	ReasonSkippedCI      Reason = "skipped_ci"
	ReasonSkippedOptOut  Reason = "skipped_optout"
	ReasonGateOff        Reason = "gate_off"
	ReasonSkippedHoldout Reason = "skipped_holdout"

	// Install, or update an existing installation.
	ReasonRun Reason = "run"
)

// Due reports whether the run acts on the decision: it installs, or it
// settles the version. A run that is not due changes nothing.
func (r Reason) Due() bool {
	switch r {
	case ReasonSkippedCommand, ReasonOnRuntime, ReasonCacheDisabled, ReasonDevelopmentBuild, ReasonRootInUserHome, ReasonAlreadyRan, ReasonRetriesExhausted, ReasonRetryNotDue, ReasonNoConsent:
		return false
	case ReasonSkippedCI, ReasonSkippedOptOut, ReasonGateOff, ReasonSkippedHoldout, ReasonRun:
		return true
	}
	return true
}

// Inputs is everything Decide depends on.
type Inputs struct {
	// SkippedCommand is set for a command that never triggers auto-install.
	SkippedCommand bool

	// OnRuntime is set on Databricks Runtime, where the home directory does
	// not outlive the cluster and the CLI version is the platform's.
	OnRuntime bool

	// CacheDisabled is DATABRICKS_CACHE_ENABLED=false.
	CacheDisabled bool

	// Version is the running CLI version.
	Version string

	// RootInUserHome is set when the CLI runs as root in a home directory
	// that another user owns (see RootInUserHome).
	RootInUserHome bool

	// State is the persisted state: last version, attempts and the user's
	// consent.
	State State

	// Now is the time of the run.
	Now time.Time

	// CI is set when the CLI runs in continuous integration.
	CI bool

	// EnvOptOut is DATABRICKS_AITOOLS_AUTO_INSTALL=false, or a "no" earlier in
	// this process (see Declined).
	EnvOptOut bool

	// AgreedNow is set when the user agreed earlier in this process, at the
	// question at login. A run that settled Version, or failed to look the
	// gate up for it, before the user agreed installed nothing, so it does not
	// stop the install the user is waiting for.
	AgreedNow bool

	// GateOn is the remote gate for Version.
	GateOn bool

	// Bucket is the machine's stable holdout bucket, in [0, 100).
	Bucket int
}

// Decide returns whether this run should install, settle the version, or do
// nothing. The checks run from cheapest to most expensive, so callers can
// leave GateOn false and look the gate up only when Decide returns
// ReasonGateOff. Every skip reason precedes the gate except the holdout:
// the holdout arm only exists once the feature is on.
func Decide(in Inputs) Reason {
	switch {
	case in.SkippedCommand:
		return ReasonSkippedCommand
	case in.OnRuntime:
		return ReasonOnRuntime
	case in.CacheDisabled:
		return ReasonCacheDisabled
	// Development and snapshot builds change version with every commit, so
	// they would trigger on every build.
	case build.IsDevelopmentVersion(in.Version):
		return ReasonDevelopmentBuild
	// Root under sudo with HOME kept would leave files the user cannot
	// change in the user's home.
	case in.RootInUserHome:
		return ReasonRootInUserHome
	case in.State.LastVersion == in.Version && in.State.Attempts >= MaxAttempts && (!in.AgreedNow || !in.State.gateLookupFailed(in.Version)):
		return ReasonRetriesExhausted
	case in.State.LastVersion == in.Version && in.State.Attempts == 0 && !in.AgreedNow:
		return ReasonAlreadyRan
	// A newer CLI already ran here, e.g. an IDE ships its own older CLI next
	// to the user's: alternating between the two must not run on every
	// command.
	case in.State.LastVersion != "" && semver.Compare("v"+in.Version, "v"+in.State.LastVersion) < 0:
		return ReasonAlreadyRan
	case in.CI:
		return ReasonSkippedCI
	case in.EnvOptOut, in.State.Consent() == ConsentOptedOut:
		return ReasonSkippedOptOut
	// Without the user's yes nothing is installed. CI and the opt-out come
	// first, so they settle the version and are reported whatever the consent.
	case in.State.Consent() == ConsentUnknown:
		return ReasonNoConsent
	// A source or agent CLI that stays broken would otherwise make every
	// following command wait for a failing attempt. The wait is for the
	// install only: CI and the opt-out settle the version during it.
	case in.State.LastVersion == in.Version && in.State.Attempts > 0 && !retryDue(in.State, in.Now):
		return ReasonRetryNotDue
	case !in.GateOn:
		return ReasonGateOff
	case InHoldout(in.Bucket):
		return ReasonSkippedHoldout
	}
	return ReasonRun
}

// retryDue reports whether the wait after the failed attempts of s is over at
// now: firstRetryDelay after the first, doubling with each one after it. A
// state without State.LastAttempt (one an older CLI wrote) is due. So is one
// whose last attempt is later than now: the clock was set back, and the wait
// would otherwise last until the clock caught up.
func retryDue(s State, now time.Time) bool {
	since := now.Sub(s.LastAttempt)
	return since < 0 || since >= firstRetryDelay<<(s.Attempts-1)
}

// gateLookupFailed reports whether s records a run for version whose gate
// lookup failed: it used up the attempts of version without starting an
// install, so it has no State.LastAttempt.
func (s State) gateLookupFailed(version string) bool {
	return s.LastVersion == version && s.Attempts >= MaxAttempts && s.LastAttempt.IsZero()
}

// Eligible reports whether a run with in may auto-install at all: neither the
// command, Databricks Runtime, a turned off cache, a development build, root
// in another user's home, CI nor an opt-out in the environment rules it out.
// It ignores in.State, the gate and the holdout.
func Eligible(in Inputs) bool {
	// Without a state, Decide stops at the first of these checks that rules
	// the run out, or else at the consent that nobody gave.
	in.State = State{}
	return Decide(in) == ReasonNoConsent
}

// InHoldout reports whether a machine in bucket belongs to the holdout.
func InHoldout(bucket int) bool {
	return bucket < HoldoutPercent
}
