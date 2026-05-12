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

// TestPromptBaseline_CtrlJ pins that Ctrl+J submits the prompt in
// [cmdio.RunPrompt] — the same as the Enter (Return) key.
//
// Enter sends CR (0x0d) and Ctrl+J sends LF (0x0a). chzyer/readline maps
// both to CharEnter and ends the read loop, so promptui returns the
// current buffer either way. A future hand-rolled prompt that only
// reacts to CR would silently swallow Ctrl+J; this test pins the parity.
func TestPromptBaseline_CtrlJ(t *testing.T) {
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

	tm.Type(termtest.KeyCtrlJ)
	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "hello", res.value)
}
