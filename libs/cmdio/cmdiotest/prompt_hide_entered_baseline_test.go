package cmdiotest_test

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_HideEnteredFalse pins the default post-Enter rendering
// of [cmdio.RunPrompt]: with HideEntered=false (the default), the entered
// value is shown alongside the label after the prompt closes.
func TestPromptBaseline_HideEnteredFalse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty-based prompt tests are unix-only")
	}

	tm := termtest.New(t)
	defer tm.Close()

	pts := tm.Pty()
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	ctx := t.Context()
	io := cmdio.NewIO(ctx, flags.OutputText, pts, pts, pts, "", "")
	ctx = cmdio.InContext(ctx, io)

	require.True(t, cmdio.IsPromptSupported(ctx), "prompt support must be detected on the pty")

	type result struct {
		value string
		err   error
	}
	resCh := make(chan result, 1)
	go func() {
		v, err := cmdio.RunPrompt(ctx, cmdio.PromptOptions{
			Label:       "Workspace name",
			HideEntered: false,
		})
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Workspace name")
	tm.Type("hello")
	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hello", res.value, "snapshot:\n%s", tm.Snapshot())

	tm.Golden("01-after-enter")
}

// TestPromptBaseline_HideEnteredTrue pins that HideEntered=true clears the
// prompt frame after the user submits, leaving no trace of the entered value
// on screen. This is the path used by [cmdio.Secret].
func TestPromptBaseline_HideEnteredTrue(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty-based prompt tests are unix-only")
	}

	tm := termtest.New(t)
	defer tm.Close()

	pts := tm.Pty()
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	ctx := t.Context()
	io := cmdio.NewIO(ctx, flags.OutputText, pts, pts, pts, "", "")
	ctx = cmdio.InContext(ctx, io)

	require.True(t, cmdio.IsPromptSupported(ctx), "prompt support must be detected on the pty")

	type result struct {
		value string
		err   error
	}
	resCh := make(chan result, 1)
	go func() {
		v, err := cmdio.RunPrompt(ctx, cmdio.PromptOptions{
			Label:       "Workspace name",
			HideEntered: true,
		})
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Workspace name")
	tm.Type("hello")
	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hello", res.value, "snapshot:\n%s", tm.Snapshot())

	tm.Golden("01-after-enter")
}
