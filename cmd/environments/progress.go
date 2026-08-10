package environments

import (
	"context"

	"github.com/databricks/cli/libs/cmdio"
	libslocalenv "github.com/databricks/cli/libs/localenv"
)

// phaseMessages maps each pipeline phase to the live-progress text shown on the
// spinner. Phases without a user-facing message (they complete instantly) are
// omitted and leave the current message in place.
var phaseMessages = map[libslocalenv.PhaseName]string{
	libslocalenv.PhasePreflight: "Checking your project…",
	libslocalenv.PhaseResolve:   "Resolving your Databricks compute…",
	libslocalenv.PhaseFetch:     "Fetching matching versions and constraints…",
	libslocalenv.PhaseMerge:     "Updating pyproject.toml…",
	libslocalenv.PhaseProvision: "Provisioning the virtual environment with uv…",
	libslocalenv.PhaseValidate:  "Validating the environment…",
}

// spinnerReporter renders live per-phase progress on a cmdio spinner. The spinner
// degrades to a no-op in non-interactive terminals (CI, acceptance tests), so
// this writes nothing there.
type spinnerReporter struct {
	sp interface {
		Update(string)
		Close()
	}
}

// newSpinnerReporter starts a spinner and returns a Reporter that updates it as
// phases begin. The caller must Close it when the run finishes.
func newSpinnerReporter(ctx context.Context) *spinnerReporter {
	return &spinnerReporter{sp: cmdio.NewSpinner(ctx)}
}

// PhaseStarted updates the spinner message for the phase that is beginning.
func (r *spinnerReporter) PhaseStarted(name libslocalenv.PhaseName) {
	if msg, ok := phaseMessages[name]; ok {
		r.sp.Update(msg)
	}
}

// Close stops the spinner.
func (r *spinnerReporter) Close() { r.sp.Close() }
