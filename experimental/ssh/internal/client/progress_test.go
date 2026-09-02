package client

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestProgressUI returns a progressUI writing to w with sentinel check/cross
// markers so the emitted outcome line is assertable. runStep is driven with a
// non-interactive cmdio context (the mode tests run in), where the spinner
// degrades to no output, exercising the capture-and-dump logic directly.
func newTestProgressUI(w io.Writer) *progressUI {
	return &progressUI{w: w, check: "OK", cross: "FAIL"}
}

func TestRunStepHidesOutputOnSuccess(t *testing.T) {
	var w bytes.Buffer
	ui := newTestProgressUI(&w)

	err := ui.runStep(cmdio.MockDiscard(t.Context()), "Installing dependencies", func(out io.Writer) error {
		_, _ = io.WriteString(out, "verbose installer chatter\n")
		return nil
	})
	require.NoError(t, err)

	// The step's subprocess output must not surface on success, but the checkmark
	// line for the step is still emitted.
	assert.NotContains(t, w.String(), "verbose installer chatter")
	assert.Contains(t, w.String(), "OK Installing dependencies")
}

func TestRunStepShowsOutputOnFailure(t *testing.T) {
	var w bytes.Buffer
	ui := newTestProgressUI(&w)

	sentinel := errors.New("install failed")
	err := ui.runStep(cmdio.MockDiscard(t.Context()), "Installing ucode", func(out io.Writer) error {
		_, _ = io.WriteString(out, "line to stdout\nline to stderr")
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)

	// On failure the cross line and the full captured output are printed, with a
	// trailing newline added.
	assert.Contains(t, w.String(), "FAIL Installing ucode")
	assert.Contains(t, w.String(), "line to stdout\nline to stderr\n")
}
