package cmdiotest_test

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_Scroll pins the current promptui scrolling behavior for a
// list larger than promptui's default visible window. It feeds enough KeyDown
// presses to reach the last item and then keeps pressing past it, so the
// goldens capture both the bottom-of-list state and the past-bottom state.
//
// This baseline lets the upcoming bubbletea reimplementation be diffed against
// the exact rendering promptui produces today. Skipped on Windows because the
// pty harness is unix-only.
func TestSelectBaseline_Scroll(t *testing.T) {
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

	items := make([]cmdio.Tuple, 0, 12)
	for i := 1; i <= 12; i++ {
		items = append(items, cmdio.Tuple{
			Name: fmt.Sprintf("item-%02d", i),
			Id:   fmt.Sprintf("id%02d", i),
		})
	}

	type result struct {
		id  string
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		id, err := cmdio.SelectOrdered(ctx, items, "Pick one")
		resCh <- result{id: id, err: err}
	}()

	tm.WaitFor("Pick one")
	tm.WaitFor("item-01")
	tm.Golden("01-initial")

	for range 11 {
		tm.Type(termtest.KeyDown)
	}
	tm.Golden("02-bottom")

	for range 5 {
		tm.Type(termtest.KeyDown)
	}
	tm.Golden("03-past-bottom")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
}
