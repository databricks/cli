package environments

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	libslocalenv "github.com/databricks/cli/libs/localenv"
)

// phaseMessages maps each pipeline phase to the live-progress text shown on the
// spinner. Phases without a user-facing message (they complete instantly) are
// omitted and leave the current message in place.
//
// PhasePreflight is intentionally absent: preflight can prompt interactively
// (confirmUvInstall via cmdio.AskYesOrNo when uv is not installed), and the
// prompt writes to the same stderr stream the spinner repaints. A spinner
// running during preflight would clobber the "[y/N]" line and make the command
// look hung. Omitting preflight keeps the spinner from starting until resolve,
// after the prompt has been answered (see spinnerReporter's lazy start).
var phaseMessages = map[libslocalenv.PhaseName]string{
	libslocalenv.PhaseResolve:   "Resolving your Databricks compute…",
	libslocalenv.PhaseFetch:     "Fetching matching versions and constraints…",
	libslocalenv.PhaseMerge:     "Updating pyproject.toml…",
	libslocalenv.PhaseProvision: "Provisioning the virtual environment with uv…",
	libslocalenv.PhaseValidate:  "Validating the environment…",
}

// progressSpinner is the subset of cmdio's spinner the reporter drives. It is an
// interface so tests can substitute a fake and assert lazy start.
type progressSpinner interface {
	Update(string)
	Close()
}

// spinnerReporter renders live per-phase progress on a cmdio spinner. The spinner
// degrades to a no-op in non-interactive terminals (CI, acceptance tests), so
// this writes nothing there.
//
// The spinner is started lazily, on the first phase that has a message, rather
// than eagerly in the constructor. This keeps its Bubble Tea program — which
// repaints stderr at ~5fps — from running during PhasePreflight, where an
// interactive uv-install prompt may be waiting on the same stream. See
// phaseMessages for why preflight has no message.
type spinnerReporter struct {
	// newSpinner constructs the spinner on first use; a field so tests inject a fake.
	newSpinner func() progressSpinner
	sp         progressSpinner
}

// newSpinnerReporter returns a Reporter that starts a spinner on the first phase
// with a message and updates it as later phases begin. The caller must Close it
// when the run finishes.
func newSpinnerReporter(ctx context.Context) *spinnerReporter {
	return &spinnerReporter{newSpinner: func() progressSpinner { return cmdio.NewSpinner(ctx) }}
}

// PhaseStarted updates the spinner message for the phase that is beginning,
// starting the spinner on first use. Phases without a message (preflight) are a
// no-op, so the spinner never runs while preflight's prompt may be active.
func (r *spinnerReporter) PhaseStarted(name libslocalenv.PhaseName) {
	msg, ok := phaseMessages[name]
	if !ok {
		return
	}
	if r.sp == nil {
		r.sp = r.newSpinner()
	}
	r.sp.Update(msg)
}

// Close stops the spinner if it was started.
func (r *spinnerReporter) Close() {
	if r.sp != nil {
		r.sp.Close()
	}
}
