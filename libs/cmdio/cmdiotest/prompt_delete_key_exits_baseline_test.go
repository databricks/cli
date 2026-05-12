package cmdiotest_test

import (
	"errors"
	"io"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_DeleteKeyExits pins a surprising behavior of
// [cmdio.RunPrompt]: pressing the Delete key (\x1b[3~) exits the prompt with
// io.EOF, just like Ctrl+D would on an empty line.
//
// Why: chzyer/readline maps both Delete and Ctrl+D to the same internal rune
// (CharDelete = 4). Its CharDelete handler treats a non-empty buffer as
// "forward-delete" and an empty buffer as EOF. Promptui's listener resets
// readline's buffer to empty after every keystroke (cur is the source of
// truth, not o.buf), so from readline's perspective the buffer is always
// empty when Delete arrives — every Delete press takes the EOF path.
//
// This test pins the current behavior so that any change (e.g. a promptui or
// readline upgrade that splits the two keys) is intentional.
func TestPromptBaseline_DeleteKeyExits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty-based prompt tests are unix-only")
	}

	tm := termtest.New(t)
	defer tm.Close()

	pts := tm.Pty()
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	ctx := t.Context()
	io2 := cmdio.NewIO(ctx, flags.OutputText, pts, pts, pts, "", "")
	ctx = cmdio.InContext(ctx, io2)

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

	// Type some content first to prove the buffer is non-empty from the user's
	// perspective. This is what makes the behavior surprising: the prompt
	// still exits even though the user has typed input.
	tm.Type("hello")
	tm.Type(termtest.KeyDelete)

	res := <-resCh
	assert.Empty(t, res.value, "Delete-as-EOF discards typed input")
	assert.Truef(t, errors.Is(res.err, io.EOF) || res.err.Error() == "^D",
		"Delete should exit with EOF; got err=%v (raw output: %q)", res.err, tm.Raw())
}
