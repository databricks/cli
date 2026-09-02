package client

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestProgressUI returns a progressUI writing to w with a non-interactive
// cmdio context (the mode tests run in, no TTY). The spinner degrades to no
// output there, so this exercises runStep's capture-and-dump logic directly.
func newTestProgressUI(ctx context.Context, w io.Writer) *progressUI {
	return &progressUI{ctx: cmdio.MockDiscard(ctx), w: w}
}

func TestRunStepHidesOutputOnSuccess(t *testing.T) {
	var w bytes.Buffer
	ui := newTestProgressUI(t.Context(), &w)

	err := ui.runStep("Installing dependencies", func(out io.Writer) error {
		_, _ = io.WriteString(out, "verbose installer chatter\n")
		return nil
	})
	require.NoError(t, err)

	// The step's subprocess output must not surface on success.
	assert.NotContains(t, w.String(), "verbose installer chatter")
}

func TestRunStepShowsOutputOnFailure(t *testing.T) {
	var w bytes.Buffer
	ui := newTestProgressUI(t.Context(), &w)

	sentinel := errors.New("install failed")
	err := ui.runStep("Installing ucode", func(out io.Writer) error {
		_, _ = io.WriteString(out, "line to stdout\nline to stderr")
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)

	// On failure the full captured output is printed, with a trailing newline added.
	assert.Contains(t, w.String(), "line to stdout\nline to stderr\n")
}
