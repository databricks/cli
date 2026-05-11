package cmdiotest_test

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_DefaultNoEdit pins Prompt's behavior when Default is
// set but AllowEdit is left at its zero value (false). The default renders
// as a hint rather than pre-filling the buffer, so the first keystroke
// replaces it instead of concatenating.
func TestPromptBaseline_DefaultNoEdit(t *testing.T) {
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
			Label:   "Region",
			Default: "us-west-2",
		})
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Region")
	tm.Golden("01-default-shown")

	tm.Type("e")
	tm.Golden("02-after-one-char")

	tm.Type("u-central-1")
	tm.Golden("03-after-typing")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	t.Logf("returned: %q", res.value)
}
