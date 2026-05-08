package cmdiotest_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_CtrlC pins the current promptui-driven Select behavior
// when the user cancels the prompt with Ctrl+C without making a selection.
// Captured as a migration baseline for the upcoming bubbletea replacement.
func TestSelectBaseline_CtrlC(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pty-based prompt tests are unix-only")
	}

	tm := termtest.New(t)
	defer tm.Close()

	pts := tm.Pty()
	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")

	ctx := context.Background()
	io := cmdio.NewIO(ctx, flags.OutputText, pts, pts, pts, "", "")
	ctx = cmdio.InContext(ctx, io)

	require.True(t, cmdio.IsPromptSupported(ctx), "prompt support must be detected on the pty")

	type result struct {
		id  string
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		id, err := cmdio.SelectOrdered(ctx, []cmdio.Tuple{
			{Name: "alpha", Id: "a"},
			{Name: "beta", Id: "b"},
			{Name: "gamma", Id: "g"},
		}, "Pick one")
		resCh <- result{id: id, err: err}
	}()

	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyCtrlC)

	res := <-resCh
	require.Error(t, res.err)
	t.Logf("error: %v", res.err)
	t.Logf("id: %q", res.id)
}
