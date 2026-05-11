package cmdiotest_test

import (
	"context"
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSelectBaseline_CtrlJ pins promptui Select's surprising response to
// Ctrl+J. Ctrl+J sends LF (0x0a) and Enter sends CR (0x0d). In a vacuum,
// chzyer/readline maps both to CharEnter — but inside promptui's Select
// loop they don't behave identically. Ctrl+J does end the prompt cleanly
// (no error), but the highlighted item is reset to the first one before
// the result is returned: pressing Down to move onto "beta" and then
// Ctrl+J yields "a", not "b".
//
// Plain Enter from the same state correctly returns "b" — that path is
// pinned by TestSelectBaseline_DownEnter — so the divergence is real and
// specific to LF. The baseline locks the current behaviour so a future
// hand-rolled Select can decide whether to preserve, fix, or fail loudly
// on it.
func TestSelectBaseline_CtrlJ(t *testing.T) {
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

	tm.Type(termtest.KeyDown)
	tm.Golden("02-after-down")

	tm.Type(termtest.KeyCtrlJ)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	// Highlighted item before Ctrl+J was beta; the returned id is alpha.
	// This is the parity miss the test is here to pin.
	assert.Equal(t, "a", res.id, "snapshot:\n%s", tm.Snapshot())
}
