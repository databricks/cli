package cmdiotest_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// step is one keystroke + the label used for its golden snapshot.
type step struct {
	label string
	input string
}

// runDefaultedPrompt runs RunPrompt with the given Default value in a pty,
// sends each step's input, records a golden snapshot after each step,
// then submits with Enter and returns the prompt's value.
func runDefaultedPrompt(t *testing.T, defaultVal string, steps []step) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("pty-based prompt tests are unix-only")
	}

	tm := termtest.New(t)
	t.Cleanup(tm.Close)

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
			Default: defaultVal,
		})
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Region")
	tm.Golden("00-default-shown")

	for i, s := range steps {
		tm.Type(s.input)
		tm.Golden(fmt.Sprintf("%02d-%s", i+1, s.label))
	}
	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	return res.value
}

// TestPromptBaseline_DefaultBackspaceErases pins that the first Backspace
// on a pending default discards the entire default value (promptui's
// eraseDefault path). After the backspace the buffer is empty; pressing
// Enter submits "".
func TestPromptBaseline_DefaultBackspaceErases(t *testing.T) {
	v := runDefaultedPrompt(t, "us-west-2", []step{
		{"after-backspace", termtest.KeyBackspace},
	})
	assert.Equal(t, "", v)
}

// TestPromptBaseline_DefaultCtrlHErases pins that Ctrl+H behaves the same
// as Backspace on a pending default: it discards the entire default.
func TestPromptBaseline_DefaultCtrlHErases(t *testing.T) {
	v := runDefaultedPrompt(t, "us-west-2", []step{
		{"after-ctrl-h", termtest.KeyCtrlH},
	})
	assert.Equal(t, "", v)
}

// TestPromptBaseline_DefaultLeftIsInert pins that the first Left arrow on
// a pending default does not opt into editing the default (only Right /
// Ctrl+F does that — see TestPromptBaseline_DefaultCursorMovementEntersBuffer).
// The cursor is already at column 0 so Left is visually a no-op, and the
// next printable rune still replaces the default rather than inserting
// before it.
func TestPromptBaseline_DefaultLeftIsInert(t *testing.T) {
	v := runDefaultedPrompt(t, "us-west-2", []step{
		{"after-left", termtest.KeyLeft},
		{"after-typing", "x"},
	})
	assert.Equal(t, "x", v)
}

// TestPromptBaseline_DefaultCtrlBIsInert pins that Ctrl+B behaves the same
// as Left on a pending default.
func TestPromptBaseline_DefaultCtrlBIsInert(t *testing.T) {
	v := runDefaultedPrompt(t, "us-west-2", []step{
		{"after-ctrl-b", termtest.KeyCtrlB},
		{"after-typing", "x"},
	})
	assert.Equal(t, "x", v)
}

// TestPromptBaseline_DefaultRightThenBackspace pins what happens once
// Right has loaded the default into the editable buffer: Backspace now
// deletes the character to the left of the cursor (within the loaded
// default), not the entire default. After Right + Backspace the buffer
// is "s-west-2".
func TestPromptBaseline_DefaultRightThenBackspace(t *testing.T) {
	v := runDefaultedPrompt(t, "us-west-2", []step{
		{"after-right", termtest.KeyRight},
		{"after-backspace", termtest.KeyBackspace},
	})
	assert.Equal(t, "s-west-2", v)
}
