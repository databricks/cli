package environments

import (
	"testing"

	libslocalenv "github.com/databricks/cli/libs/localenv"
	"github.com/stretchr/testify/assert"
)

// fakeSpinner records the messages and Close calls the reporter makes.
type fakeSpinner struct {
	updates []string
	closed  int
}

func (f *fakeSpinner) Update(msg string) { f.updates = append(f.updates, msg) }
func (f *fakeSpinner) Close()            { f.closed++ }

// newTestReporter builds a spinnerReporter whose spinner is a fake, and returns
// both plus a counter of how many times a spinner was constructed.
func newTestReporter() (*spinnerReporter, *fakeSpinner, *int) {
	sp := &fakeSpinner{}
	created := 0
	r := &spinnerReporter{newSpinner: func() progressSpinner {
		created++
		return sp
	}}
	return r, sp, &created
}

// Preflight has no message, so the reporter must not construct a spinner while
// preflight's interactive uv-install prompt may be on the same stream.
func TestSpinnerReporterDoesNotStartOnPreflight(t *testing.T) {
	r, sp, created := newTestReporter()

	r.PhaseStarted(libslocalenv.PhasePreflight)

	assert.Equal(t, 0, *created, "spinner must not start during preflight")
	assert.Empty(t, sp.updates)

	// Close before any message-bearing phase is a no-op (nothing to stop).
	r.Close()
	assert.Equal(t, 0, sp.closed)
}

// The spinner starts on the first message-bearing phase and is reused (not
// reconstructed) for later phases; Close stops it once.
func TestSpinnerReporterStartsLazilyAndReuses(t *testing.T) {
	r, sp, created := newTestReporter()

	r.PhaseStarted(libslocalenv.PhasePreflight) // no-op
	r.PhaseStarted(libslocalenv.PhaseResolve)   // starts the spinner
	r.PhaseStarted(libslocalenv.PhaseFetch)     // reuses it

	assert.Equal(t, 1, *created, "spinner should be constructed exactly once")
	assert.Equal(t, []string{
		phaseMessages[libslocalenv.PhaseResolve],
		phaseMessages[libslocalenv.PhaseFetch],
	}, sp.updates)

	r.Close()
	assert.Equal(t, 1, sp.closed)
}
