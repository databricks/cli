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

// TestPromptBaseline_CtrlFCtrlB pins that Ctrl+F and Ctrl+B move the cursor
// one character forward and backward in [cmdio.RunPrompt], the same as the
// right and left arrow keys.
//
// chzyer/readline maps Ctrl+F to CharForward and Ctrl+B to CharBackward —
// the same runes the arrow keys decode to — and promptui's Cursor.Listen
// dispatches both via its KeyForward / KeyBackward cases. So the emacs-
// style bindings are de-facto aliases for the arrow keys; this test pins
// that equivalence.
func TestPromptBaseline_CtrlFCtrlB(t *testing.T) {
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
			Label: "Workspace name",
		})
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Workspace name")
	tm.Type("hello")
	tm.Golden("01-cursor-end")

	tm.Type(termtest.KeyCtrlB)
	tm.Type(termtest.KeyCtrlB)
	tm.Golden("02-after-ctrl-b-twice")

	tm.Type(termtest.KeyCtrlF)
	tm.Golden("03-after-ctrl-f")

	tm.Type(termtest.KeyEnter)
	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hello", res.value)
}
