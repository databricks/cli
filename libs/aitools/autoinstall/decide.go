package autoinstall

import (
	"slices"

	"github.com/databricks/cli/internal/build"
	"golang.org/x/mod/semver"
)

const (
	// MaxAttempts is how many failed attempts one CLI version gets. A new
	// version starts over.
	MaxAttempts = 10

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
	ReasonCacheDisabled    Reason = "cache_disabled"
	ReasonDevelopmentBuild Reason = "development_build"
	ReasonAlreadyRan       Reason = "already_ran"
	ReasonRetriesExhausted Reason = "retries_exhausted"

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
	case ReasonSkippedCommand, ReasonCacheDisabled, ReasonDevelopmentBuild, ReasonAlreadyRan, ReasonRetriesExhausted:
		return false
	case ReasonSkippedCI, ReasonSkippedOptOut, ReasonGateOff, ReasonSkippedHoldout, ReasonRun:
		return true
	}
	return true
}

// Inputs is everything Decide depends on.
type Inputs struct {
	// Command is the command path below the root, e.g. ["auth", "token"].
	Command []string

	// CacheDisabled is DATABRICKS_CACHE_ENABLED=false.
	CacheDisabled bool

	// Version is the running CLI version.
	Version string

	// State is the persisted state: last version, attempts and opt-out.
	State State

	// CI is the CI env var check of the version-check notice.
	CI bool

	// EnvOptOut is DATABRICKS_AITOOLS_AUTO_INSTALL=false, or a "no" earlier in
	// this process (see Declined).
	EnvOptOut bool

	// GateOn is the remote gate for Version.
	GateOn bool

	// Bucket is the machine's stable holdout bucket, in [0, 100).
	Bucket int
}

// skippedCommands never trigger auto-install anywhere in their path: help
// and shell completion output is read by machines or shown on every
// keystroke, `version` is how a user checks what was just installed, and the
// aitools subtree manages the tools explicitly. "__complete" and
// "__completeNoDesc" are cobra.ShellCompRequestCmd and
// cobra.ShellCompNoDescRequestCmd.
var skippedCommands = []string{"aitools", "help", "version", "completion", "__complete", "__completeNoDesc"}

// Decide returns whether this run should install, settle the version, or do
// nothing. The checks run from cheapest to most expensive, so callers can
// leave GateOn false and look the gate up only when Decide returns
// ReasonGateOff. Every skip reason precedes the gate except the holdout:
// the holdout arm only exists once the feature is on.
func Decide(in Inputs) Reason {
	switch {
	case isSkippedCommand(in.Command):
		return ReasonSkippedCommand
	case in.CacheDisabled:
		return ReasonCacheDisabled
	// Development and snapshot builds change version with every commit, so
	// they would trigger on every build.
	case build.IsDevelopmentVersion(in.Version):
		return ReasonDevelopmentBuild
	case in.State.LastVersion == in.Version && in.State.Attempts >= MaxAttempts:
		return ReasonRetriesExhausted
	case in.State.LastVersion == in.Version && in.State.Attempts == 0:
		return ReasonAlreadyRan
	// A newer CLI already ran here, e.g. an IDE ships its own older CLI next
	// to the user's: alternating between the two must not run on every
	// command.
	case in.State.LastVersion != "" && semver.Compare("v"+in.Version, "v"+in.State.LastVersion) < 0:
		return ReasonAlreadyRan
	case in.CI:
		return ReasonSkippedCI
	case in.EnvOptOut, in.State.OptedOut:
		return ReasonSkippedOptOut
	case !in.GateOn:
		return ReasonGateOff
	case InHoldout(in.Bucket):
		return ReasonSkippedHoldout
	}
	return ReasonRun
}

// InHoldout reports whether a machine in bucket belongs to the holdout.
func InHoldout(bucket int) bool {
	return bucket < HoldoutPercent
}

func isSkippedCommand(path []string) bool {
	for _, name := range path {
		if slices.Contains(skippedCommands, name) {
			return true
		}
	}
	return slices.Equal(path, []string{"auth", "token"})
}
