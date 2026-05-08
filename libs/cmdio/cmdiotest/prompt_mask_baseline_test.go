package cmdiotest_test

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_Mask pins runprompts behavior
// when configured with Mask='*'. This is the shape used by `databricks
// configure` for personal access token entry (cmd/configure/configure.go:46).
func TestPromptBaseline_Mask(t *testing.T) {
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
			Label: "Personal access token",
			Mask:  '*',
		})
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Personal access token")
	tm.Golden("01-empty")

	tm.Type("dapi-secret")
	tm.Golden("02-after-typing")

	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Type(termtest.KeyBackspace)
	tm.Golden("03-after-backspace")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "dapi-sec", res.value, "snapshot:\n%s", tm.Snapshot())
}
