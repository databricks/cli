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

// TestPromptBaseline_CtrlH pins that Ctrl+H deletes the character to the
// left of the cursor in [cmdio.RunPrompt] — the same as the Backspace key.
//
// Ctrl+H sends BS (0x08) and Backspace sends DEL (0x7f). chzyer/readline
// maps both to CharBackspace, and promptui's Cursor.Listen handles them
// uniformly via its KeyBackspace case. So the control-character form is
// a de-facto alias for the Backspace key; this test pins that equivalence
// so a future hand-rolled prompt implementation can't silently drop it.
func TestPromptBaseline_CtrlH(t *testing.T) {
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
	tm.Golden("01-typed-hello")

	tm.Type(termtest.KeyCtrlH)
	tm.Type(termtest.KeyCtrlH)
	tm.Golden("02-after-ctrl-h-twice")

	tm.Type(termtest.KeyEnter)
	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hel", res.value)
}
