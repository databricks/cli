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

// TestSelectBaseline_FilterScroll pins viewport behavior when a filter
// narrows a long list to a count still larger than the viewport. Combines
// FilterTyping (substring search) with Scroll (12+ items) — neither test
// alone exercises the recompute-then-scroll path.
//
// 20 items named item-01 .. item-20; the filter "item-1" matches item-01
// plus item-10..item-19 = 11 items, more than the 5-row viewport.
func TestSelectBaseline_FilterScroll(t *testing.T) {
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

	items := make([]cmdio.Tuple, 0, 20)
	for i := 1; i <= 20; i++ {
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

	tm.Type("item-1")
	tm.Golden("02-filtered-top")

	for range 5 {
		tm.Type(termtest.KeyDown)
	}
	tm.Golden("03-filtered-mid")

	for range 10 {
		tm.Type(termtest.KeyDown)
	}
	tm.Golden("04-filtered-bottom")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	t.Logf("selected: %s", res.id)
}
