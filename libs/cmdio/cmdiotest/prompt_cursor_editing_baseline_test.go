package cmdiotest_test

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_CursorEditing pins how RunPrompt responds to cursor
// movement and line-editing keys: ←/→, Home/End, Backspace, Ctrl+W, Ctrl+U.
// Promptui's Cursor.Listen handles Backspace; Ctrl+W and Ctrl+U have no case
// there, so they're no-ops in this prompt — the goldens after them are
// intentionally identical to the post-Backspace one. The Delete key (\x1b[3~)
// is *not* covered here because it conflates with Ctrl+D in chzyer/readline
// and exits the prompt; that behavior is pinned separately by
// TestPromptBaseline_DeleteKeyExits.
func TestPromptBaseline_CursorEditing(t *testing.T) {
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
	tm.Golden("01-empty")

	tm.Type("hello world")
	tm.Golden("02-typed")

	tm.Type(termtest.KeyHome)
	tm.Type("X")
	tm.Golden("03-insert-at-start")

	tm.Type(termtest.KeyEnd)
	tm.Type("!")
	tm.Golden("04-insert-at-end")

	tm.Type(termtest.KeyLeft)
	tm.Type(termtest.KeyLeft)
	tm.Type("Y")
	tm.Golden("05-insert-mid")

	tm.Type(termtest.KeyBackspace)
	tm.Golden("06-after-backspace")

	tm.Type(termtest.KeyCtrlW)
	tm.Golden("07-after-ctrl-w")

	tm.Type(termtest.KeyCtrlU)
	tm.Golden("08-after-ctrl-u")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	t.Logf("returned: %q (err=%v)", res.value, res.err)
}
