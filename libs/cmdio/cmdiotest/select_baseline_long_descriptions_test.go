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

// TestSelectBaseline_LongDescriptions pins the current promptui-driven Select
// behavior when item Ids are long enough to potentially overflow the terminal
// width. The active row uses the Tuple template
// "{{.Name | bold}} ({{.Id|faint}})", so the long Id is only rendered on the
// active line; non-active rows show only the Name.
//
// This is a migration baseline: the bubbletea replacement should produce the
// same visible output, captured in goldens via vt10x.
func TestSelectBaseline_LongDescriptions(t *testing.T) {
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

	items := []cmdio.Tuple{
		{Name: "short", Id: "this-is-a-very-long-resource-identifier-that-exceeds-typical-width-1234567890"},
		{Name: "medium-length-name", Id: "another-extremely-long-id-string-with-lots-of-content-aaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{Name: "x", Id: "yet-another-long-identifier-with-quite-a-bit-of-text-bbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	}

	type result struct {
		id  string
		err error
	}
	resCh := make(chan result, 1)
	go func() {
		id, err := cmdio.SelectOrdered(ctx, items, "Pick a resource")
		resCh <- result{id: id, err: err}
	}()

	tm.WaitFor("Pick a resource")
	tm.WaitFor("short")
	tm.Golden("01-initial")

	tm.Type(termtest.KeyDown)
	tm.Golden("02-second-active")

	tm.Type(termtest.KeyDown)
	tm.Golden("03-third-active")

	tm.Type(termtest.KeyEnter)

	res := <-resCh
	require.NoError(t, res.err, "raw output: %q", tm.Raw())
	assert.Equal(t, items[2].Id, res.id, "snapshot:\n%s", tm.Snapshot())
}
