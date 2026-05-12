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

// TestPromptBaseline_AltKeyNoop pins that Alt-prefixed keys are silent
// no-ops in [cmdio.RunPrompt]. Specifically, Alt+f (the readline binding
// for "move forward by word") must neither move the cursor nor insert a
// literal 'f' into the buffer.
//
// Why: chzyer/readline does process Alt+f — it calls o.buf.MoveToNextWord
// and fires the listener with key=MetaForward — but promptui's
// Cursor.Listen has no case for MetaForward and falls to a default branch
// that only does anything in erase-default mode. The listener wrapper
// then returns (nil, 0, true), which makes readline overwrite its buffer
// with empty. Net effect on the user-visible state (promptui's own
// `cur`): nothing changes.
//
// The same shape applies to Alt+b, Alt+d, Alt+Backspace and any other
// modified key promptui doesn't handle. Pinning Alt+f covers the class.
func TestPromptBaseline_AltKeyNoop(t *testing.T) {
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

	// Type "hello" and move cursor two places left so it sits mid-word.
	// If Alt+f moved the cursor (or inserted), goldens 01 and 02 would
	// diverge.
	tm.Type("hello")
	tm.Type(termtest.KeyLeft)
	tm.Type(termtest.KeyLeft)
	tm.Golden("01-cursor-mid")

	tm.Type("\x1bf")
	tm.Golden("02-after-alt-f")

	tm.Type(termtest.KeyEnter)
	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	// Final guard: the returned value must be exactly "hello". A literal
	// 'f' insertion would surface here even if the goldens above somehow
	// missed it.
	assert.Equal(t, "hello", res.value)
}
