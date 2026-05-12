package cmdiotest_test

import (
	"runtime"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPromptBaseline_UTF8 pins multi-byte rune handling: typing "café"
// (4 runes, 5 bytes) renders as 4 cells, one Backspace deletes one rune
// not one byte, and the returned value preserves the original code points.
//
// Promptui delegates to readline which is rune-aware. A bubbletea
// reimplementation that counts bytes will silently corrupt non-ASCII input
// even though ASCII tests still pass — exactly the kind of regression a
// migration baseline is meant to catch.
func TestPromptBaseline_UTF8(t *testing.T) {
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
			Label: "Name",
		})
		resCh <- result{value: v, err: err}
	}()

	tm.WaitFor("Name")
	tm.Golden("01-empty")

	tm.Type("café")
	tm.Golden("02-after-typing")

	tm.Type(termtest.KeyBackspace)
	tm.Golden("03-after-backspace")

	tm.Type("é")
	tm.Golden("04-restored")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, "café", res.value, "snapshot:\n%s", tm.Snapshot())
}
