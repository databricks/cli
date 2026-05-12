package cmdiotest_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_EscKey pins the current promptui-driven Select behavior
// when the user presses Esc at various states: the initial prompt, and after
// typing into the search filter. cmdio.Select uses StartInSearchMode: true,
// so the filter is active from the start.
//
// This test exists so the upcoming bubbletea replacement can be checked
// against a known-good baseline. Skipped on Windows because pty support there
// requires ConPTY plumbing creack/pty does not provide.
func TestSelectBaseline_EscKey(t *testing.T) {
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

	tm.Type(termtest.KeyEsc)
	tm.Golden("02-esc-from-initial")

	tm.Type("a")
	tm.Golden("03-after-typing-a")

	tm.Type(termtest.KeyEsc)
	tm.Golden("04-esc-clears-filter-or-not")

	select {
	case res := <-resCh:
		t.Logf("prompt returned after Esc: id=%q err=%v", res.id, res.err)
		t.Logf("snapshot:\n%s", tm.Snapshot())
	case <-time.After(200 * time.Millisecond):
		tm.Type(termtest.KeyEnter)
		res := <-resCh
		t.Logf("prompt finalized with Enter: id=%q err=%v", res.id, res.err)
		t.Logf("snapshot:\n%s", tm.Snapshot())
	}
}
