package cmdiotest_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_FilterTyping pins the current promptui-driven Select
// behavior when the user types letters that filter the list. cmdio.Select
// uses StartInSearchMode: true with a case-insensitive substring searcher on
// Name, so each keystroke immediately narrows the visible options.
//
// This test exists so the upcoming bubbletea replacement can be checked
// against a known-good baseline. Skipped on Windows because pty support there
// requires ConPTY plumbing creack/pty does not provide.
func TestSelectBaseline_FilterTyping(t *testing.T) {
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
			{Name: "delta", Id: "d"},
			{Name: "epsilon", Id: "e"},
		}, "Pick one")
		resCh <- result{id: id, err: err}
	}()

	tm.WaitFor("Pick one")
	tm.WaitFor("alpha")
	tm.Golden("01-initial")

	tm.Type("a")
	tm.Golden("02-after-a")

	tm.Type("l")
	tm.Golden("03-after-al")

	tm.Type("p")
	tm.Golden("04-after-alp")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "a", res.id, "snapshot:\n%s", tm.Snapshot())
}
