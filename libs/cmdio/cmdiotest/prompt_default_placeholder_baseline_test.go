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

// TestPromptBaseline_DefaultCursorMovementEntersBuffer pins the surprising
// behavior of PromptOptions.Default in promptui. The default is rendered
// like a placeholder (cursor sits at column 0 with the default text to
// its right), but the readline buffer is actually pre-filled — so the
// behavior depends on what the user does first:
//
//   - Type a rune first: the placeholder is discarded and the buffer
//     becomes just that rune (pinned by TestPromptBaseline_DefaultNoEdit).
//   - Press Right or Ctrl+F first: the cursor moves into the pre-filled
//     buffer instead of dismissing it; later typing inserts at the new
//     cursor position. After Right×2 + Ctrl+F on default "us-west-2",
//     typing "e" produces "us-ewest-2".
//
// This split is non-obvious from the visual rendering and product call
// sites don't appear to rely on it, but a future Prompt implementation
// should know it exists before changing it.
func TestPromptBaseline_DefaultCursorMovementEntersBuffer(t *testing.T) {
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
			Label:   "Region",
			Default: "us-west-2",
		})
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Region")
	tm.Golden("01-default-shown")

	// Right and Ctrl+F load the pre-filled default into the editable buffer
	// and move the cursor within it; the placeholder is not dismissed.
	tm.Type(termtest.KeyRight)
	tm.Type(termtest.KeyRight)
	tm.Type(termtest.KeyCtrlF)
	tm.Golden("02-after-right-and-ctrl-f")

	// Typing now inserts at the cursor position inside the default text.
	tm.Type("e")
	tm.Golden("03-after-typing-one-char")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "us-ewest-2", res.value, "snapshot:\n%s", tm.Snapshot())
}

// TestPromptBaseline_DefaultEnterAccepts pins that pressing Enter on an
// untouched prompt with a Default value submits the default verbatim.
func TestPromptBaseline_DefaultEnterAccepts(t *testing.T) {
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
			Label:   "Region",
			Default: "us-west-2",
		})
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Region")
	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "us-west-2", res.value, "snapshot:\n%s", tm.Snapshot())
}
