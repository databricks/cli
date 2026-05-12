package cmdiotest_test

import (
	"runtime"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
)

// TestSelectBaseline_TabKey pins the current promptui-driven Select behavior
// when the user presses Tab. Tab is a common navigation key but its handling
// in promptui's search-mode Select is unclear, so this test records the
// observed behavior as a migration baseline for the bubbletea replacement.
func TestSelectBaseline_TabKey(t *testing.T) {
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

	if !cmdio.IsPromptSupported(ctx) {
		t.Fatal("prompt support must be detected on the pty")
	}

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

	tm.Type(termtest.KeyTab)
	tm.Golden("02-after-tab")

	tm.Type(termtest.KeyTab)
	tm.Golden("03-after-second-tab")

	tm.Type(termtest.KeyEnter)

	// Enter may not terminate the prompt: in search mode with no matching
	// items, promptui's outer loop keeps calling Readline. Wait briefly, and
	// if no result arrives, send Ctrl+C so the goroutine can exit cleanly.
	// The diagnostic is what we're after — any error or selection is recorded.
	select {
	case res := <-resCh:
		t.Logf("returned id=%q err=%v", res.id, res.err)
	case <-time.After(500 * time.Millisecond):
		tm.Type(termtest.KeyCtrlC)
		res := <-resCh
		t.Logf("Enter did not terminate; after Ctrl+C: id=%q err=%v", res.id, res.err)
	}
}
