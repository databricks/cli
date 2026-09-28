package client

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunStepHidesOutputOnSuccess(t *testing.T) {
	var w bytes.Buffer
	mockCmdio := cmdio.NewIO(t.Context(), "text", io.NopCloser(strings.NewReader("")), &w, &w, "", "")

	err := runStep(cmdio.InContext(t.Context(), mockCmdio), "Installing dependencies", func(out io.Writer) error {
		_, _ = io.WriteString(out, "verbose installer chatter\n")
		return nil
	})
	require.NoError(t, err)

	// The step's subprocess output must not surface on success, but the checkmark
	// line for the step is still emitted.
	assert.NotContains(t, w.String(), "verbose installer chatter")
	assert.Contains(t, w.String(), "✓ Installing dependencies")
}

func TestRunStepShowsOutputOnFailure(t *testing.T) {
	var w bytes.Buffer
	mockCmdio := cmdio.NewIO(t.Context(), "text", io.NopCloser(strings.NewReader("")), &w, &w, "", "")

	sentinel := errors.New("install failed")
	err := runStep(cmdio.InContext(t.Context(), mockCmdio), "Installing ucode", func(out io.Writer) error {
		_, _ = io.WriteString(out, "line to stdout\nline to stderr")
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)

	// On failure the cross line and the full captured output are printed, with a
	// trailing newline added.
	assert.Contains(t, w.String(), "✗ Installing ucode")
	assert.Contains(t, w.String(), "line to stdout\nline to stderr\n")
}
