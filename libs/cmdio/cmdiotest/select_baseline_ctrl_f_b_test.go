package cmdiotest_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_CtrlFCtrlB pins that Ctrl+F and Ctrl+B page through
// the selection list — distinct from Ctrl+N / Ctrl+P which move by one
// item. Promptui maps these to KeyForward / KeyBackward (the same runes
// the right and left arrow keys decode to), and the select widget treats
// them as page-down / page-up.
//
// The list has 12 items against promptui's default visible window of 5,
// so a single Ctrl+F should advance the highlighted item by roughly a
// page rather than a single row, and Ctrl+B should walk it back.
func TestSelectBaseline_CtrlFCtrlB(t *testing.T) {
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

	tm.Type(termtest.KeyCtrlF)
	tm.Golden("02-after-ctrl-f")

	tm.Type(termtest.KeyCtrlF)
	tm.Golden("03-after-ctrl-f-twice")

	tm.Type(termtest.KeyCtrlB)
	tm.Golden("04-after-ctrl-b")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
}
